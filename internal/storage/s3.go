package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	tmtypes "github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog"

	pkgconfig "github.com/maximseshuk/snapr/internal/config"
	"github.com/maximseshuk/snapr/internal/utils"
)

type S3Storage struct {
	clients sync.Map // key string -> *s3.Client
}

func NewS3Storage() *S3Storage {
	return &S3Storage{}
}

func (s *S3Storage) GetType() string {
	return "s3"
}

// clientKey scopes a cached S3 client to one credential set — different keys must not share clients.
func clientKey(storage pkgconfig.StorageConfig) string {
	return strings.Join([]string{storage.Endpoint, storage.Region, storage.AccessKeyID, storage.SecretAccessKey}, "|")
}

func (s *S3Storage) newClient(ctx context.Context, storage pkgconfig.StorageConfig) (*s3.Client, error) {
	key := clientKey(storage)
	if cached, ok := s.clients.Load(key); ok {
		return cached.(*s3.Client), nil
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			storage.AccessKeyID,
			storage.SecretAccessKey,
			"",
		)),
		config.WithRegion(storage.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("error loading AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if storage.Endpoint != "" {
			o.BaseEndpoint = aws.String(storage.Endpoint)
			o.UsePathStyle = true
		}
	})

	actual, _ := s.clients.LoadOrStore(key, client)
	return actual.(*s3.Client), nil
}

func (s *S3Storage) jobPrefix(storage pkgconfig.StorageConfig, jobName string) string {
	base := strings.TrimSuffix(storage.Path, "/")
	seg := jobNameSegment(storage.IncludeJobName, jobName)
	switch {
	case base == "" && seg == "":
		return ""
	case base == "":
		return seg + "/"
	case seg == "":
		return base + "/"
	default:
		return base + "/" + seg + "/"
	}
}

func (s *S3Storage) EnsureJobDir(ctx context.Context, job *pkgconfig.JobConfig, storage pkgconfig.StorageConfig) error {
	_ = ctx
	_ = job
	_ = storage
	return nil
}

func (s *S3Storage) UploadInto(ctx context.Context, archivePath string, job *pkgconfig.JobConfig, wrapperRelDir string, storage pkgconfig.StorageConfig) error {
	logger := zerolog.Ctx(ctx)

	client, err := s.newClient(ctx, storage)
	if err != nil {
		return err
	}

	tm := transfermanager.New(client)

	file, err := os.Open(archivePath) //nolint:gosec // archivePath comes from snapr's own pipeline
	if err != nil {
		return fmt.Errorf("error opening archive file: %w", err)
	}
	defer func() { _ = file.Close() }()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error getting file info: %w", err)
	}

	fileName := filepath.Base(archivePath)
	prefix := s.jobPrefix(storage, job.Name)
	if wrapperRelDir != "" {
		prefix = prefix + wrapperRelDir + "/"
	}
	key := prefix + fileName

	uploadInput := &transfermanager.UploadObjectInput{
		Bucket: aws.String(storage.Bucket),
		Key:    aws.String(key),
		Body:   file,
	}

	if storage.StorageClass != "" {
		uploadInput.StorageClass = s.parseStorageClass(ctx, storage.StorageClass)
	}

	uploadStart := time.Now()
	if _, err := tm.UploadObject(ctx, uploadInput); err != nil {
		return fmt.Errorf("error uploading to S3: %w", err)
	}

	logger.Info().
		Str("file_name", fileName).
		Str("bucket", storage.Bucket).
		Str("s3_key", key).
		Int64("file_size_bytes", stat.Size()).
		Str("file_size_human", utils.FormatBytes(stat.Size())).
		Dur("upload_duration", time.Since(uploadStart)).
		Msg("Successfully uploaded archive to S3")
	return nil
}

func (s *S3Storage) ListFiles(ctx context.Context, job *pkgconfig.JobConfig, storage pkgconfig.StorageConfig) ([]FileInfo, error) {
	logger := zerolog.Ctx(ctx)

	client, err := s.newClient(ctx, storage)
	if err != nil {
		return nil, err
	}

	prefix := s.jobPrefix(storage, job.Name)

	listInput := &s3.ListObjectsV2Input{
		Bucket:    aws.String(storage.Bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	}

	var out []FileInfo
	paginator := s3.NewListObjectsV2Paginator(client, listInput)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("error listing S3 objects: %w", err)
		}
		for _, obj := range page.Contents {
			if obj.Key == nil || obj.LastModified == nil {
				continue
			}
			fileName := strings.TrimPrefix(*obj.Key, prefix)
			if fileName == "" || strings.Contains(fileName, "/") {
				continue
			}
			out = append(out, FileInfo{
				Name:         fileName,
				LastModified: *obj.LastModified,
				Size:         aws.ToInt64(obj.Size),
			})
		}
		for _, cp := range page.CommonPrefixes {
			if cp.Prefix == nil {
				continue
			}
			rel := strings.TrimSuffix(strings.TrimPrefix(*cp.Prefix, prefix), "/")
			if rel == "" {
				continue
			}
			if _, _, _, ok := ParseSplitWrapper(rel); !ok {
				continue
			}
			out = append(out, FileInfo{
				Name:    rel,
				Wrapper: true,
			})
		}
	}

	logger.Debug().
		Int("entries", len(out)).
		Str("bucket", storage.Bucket).
		Str("prefix", prefix).
		Msg("Listed S3 entries")
	return out, nil
}

func (s *S3Storage) ListWrapperParts(ctx context.Context, job *pkgconfig.JobConfig, wrapperName string, storage pkgconfig.StorageConfig) ([]FileInfo, error) {
	client, err := s.newClient(ctx, storage)
	if err != nil {
		return nil, err
	}

	prefix := s.jobPrefix(storage, job.Name) + wrapperName + "/"

	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(storage.Bucket),
		Prefix: aws.String(prefix),
	}

	var out []FileInfo
	paginator := s3.NewListObjectsV2Paginator(client, listInput)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("error listing wrapper objects: %w", err)
		}
		for _, obj := range page.Contents {
			if obj.Key == nil || obj.LastModified == nil {
				continue
			}
			fileName := strings.TrimPrefix(*obj.Key, prefix)
			if fileName == "" {
				continue
			}
			out = append(out, FileInfo{
				Name:         fileName,
				LastModified: *obj.LastModified,
				Size:         aws.ToInt64(obj.Size),
			})
		}
	}
	return out, nil
}

func (s *S3Storage) DeleteFile(ctx context.Context, job *pkgconfig.JobConfig, fileName string, storage pkgconfig.StorageConfig) error {
	logger := zerolog.Ctx(ctx)

	client, err := s.newClient(ctx, storage)
	if err != nil {
		return err
	}

	key := s.jobPrefix(storage, job.Name) + fileName
	if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(storage.Bucket),
		Key:    aws.String(key),
	}); err != nil {
		return fmt.Errorf("error deleting S3 object: %w", err)
	}

	logger.Debug().Str("s3_key", key).Msg("Deleted S3 object")
	return nil
}

func (s *S3Storage) DeleteWrapper(ctx context.Context, job *pkgconfig.JobConfig, wrapperName string, storage pkgconfig.StorageConfig) error {
	parts, err := s.ListWrapperParts(ctx, job, wrapperName, storage)
	if err != nil {
		return err
	}
	for _, p := range parts {
		key := s.jobPrefix(storage, job.Name) + wrapperName + "/" + p.Name
		client, err := s.newClient(ctx, storage)
		if err != nil {
			return err
		}
		if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(storage.Bucket),
			Key:    aws.String(key),
		}); err != nil {
			return fmt.Errorf("delete wrapper part %s: %w", key, err)
		}
	}
	zerolog.Ctx(ctx).Debug().Str("wrapper", wrapperName).Int("parts", len(parts)).Msg("Deleted S3 split wrapper")
	return nil
}

func (s *S3Storage) Download(ctx context.Context, job *pkgconfig.JobConfig, wrapperRelDir, fileName string, storage pkgconfig.StorageConfig) (*DownloadResult, error) {
	client, err := s.newClient(ctx, storage)
	if err != nil {
		return nil, err
	}

	key := s.jobPrefix(storage, job.Name)
	if wrapperRelDir != "" {
		key += wrapperRelDir + "/"
	}
	key += fileName

	if storage.DownloadMode == "signed" {
		ttl := time.Duration(storage.SignedURLTTL) * time.Second
		if ttl == 0 {
			ttl = 15 * time.Minute
		}
		presigned, perr := s3.NewPresignClient(client).PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(storage.Bucket),
			Key:    aws.String(key),
		}, s3.WithPresignExpires(ttl))
		if perr != nil {
			return nil, fmt.Errorf("presign S3 GetObject: %w", perr)
		}
		zerolog.Ctx(ctx).Debug().Str("s3_key", key).Dur("ttl", ttl).Msg("Issued S3 presigned download URL")
		return &DownloadResult{RedirectURL: presigned.URL}, nil
	}

	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(storage.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("error getting S3 object: %w", err)
	}

	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	zerolog.Ctx(ctx).Debug().Str("s3_key", key).Int64("content_length", size).Msg("Streaming download from S3")
	return &DownloadResult{Body: out.Body, Size: size}, nil
}

func (s *S3Storage) parseStorageClass(ctx context.Context, storageClass string) tmtypes.StorageClass {
	logger := zerolog.Ctx(ctx)

	switch strings.ToUpper(storageClass) {
	case "STANDARD":
		return tmtypes.StorageClassStandard
	case "REDUCED_REDUNDANCY":
		return tmtypes.StorageClassReducedRedundancy
	case "STANDARD_IA":
		return tmtypes.StorageClassStandardIa
	case "ONEZONE_IA":
		return tmtypes.StorageClassOnezoneIa
	case "INTELLIGENT_TIERING":
		return tmtypes.StorageClassIntelligentTiering
	case "GLACIER":
		return tmtypes.StorageClassGlacier
	case "DEEP_ARCHIVE":
		return tmtypes.StorageClassDeepArchive
	case "OUTPOSTS":
		return tmtypes.StorageClassOutposts
	case "GLACIER_IR":
		return tmtypes.StorageClassGlacierIr
	case "SNOW":
		return tmtypes.StorageClassSnow
	case "EXPRESS_ONEZONE":
		return tmtypes.StorageClassExpressOnezone
	default:
		logger.Warn().
			Str("storage_class", storageClass).
			Str("default_used", "STANDARD").
			Msg("Unknown storage class, using default")
		return tmtypes.StorageClassStandard
	}
}
