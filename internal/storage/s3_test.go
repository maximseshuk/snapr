package storage

import (
	"context"
	"testing"

	tmtypes "github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager/types"
	"github.com/stretchr/testify/assert"

	pkgconfig "github.com/maximseshuk/snapr/internal/config"
)

func TestS3_JobPrefix(t *testing.T) {
	s := NewS3Storage()
	cases := []struct {
		name    string
		storage pkgconfig.StorageConfig
		job     string
		want    string
	}{
		{"empty path", pkgconfig.StorageConfig{}, "myjob", "myjob/"},
		{"plain path", pkgconfig.StorageConfig{Path: "backups"}, "myjob", "backups/myjob/"},
		{"path with trailing slash", pkgconfig.StorageConfig{Path: "backups/"}, "myjob", "backups/myjob/"},
		{"nested path", pkgconfig.StorageConfig{Path: "a/b/c"}, "myjob", "a/b/c/myjob/"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, s.jobPrefix(c.storage, c.job))
		})
	}
}

func TestS3_ClientKey(t *testing.T) {
	a := clientKey(pkgconfig.StorageConfig{Endpoint: "e1", Region: "us-east-1", AccessKeyID: "AK1", SecretAccessKey: "SK1"})
	b := clientKey(pkgconfig.StorageConfig{Endpoint: "e1", Region: "us-east-1", AccessKeyID: "AK1", SecretAccessKey: "SK1"})
	assert.Equal(t, a, b, "same credentials must produce same key")

	c := clientKey(pkgconfig.StorageConfig{Endpoint: "e1", Region: "us-east-1", AccessKeyID: "AK2", SecretAccessKey: "SK1"})
	assert.NotEqual(t, a, c, "different access key must produce different cache key")

	d := clientKey(pkgconfig.StorageConfig{Endpoint: "e2", Region: "us-east-1", AccessKeyID: "AK1", SecretAccessKey: "SK1"})
	assert.NotEqual(t, a, d, "different endpoint must produce different cache key")
}

func TestS3_ParseStorageClass(t *testing.T) {
	s := NewS3Storage()
	cases := map[string]tmtypes.StorageClass{
		"STANDARD":            tmtypes.StorageClassStandard,
		"standard":            tmtypes.StorageClassStandard,
		"REDUCED_REDUNDANCY":  tmtypes.StorageClassReducedRedundancy,
		"STANDARD_IA":         tmtypes.StorageClassStandardIa,
		"ONEZONE_IA":          tmtypes.StorageClassOnezoneIa,
		"INTELLIGENT_TIERING": tmtypes.StorageClassIntelligentTiering,
		"GLACIER":             tmtypes.StorageClassGlacier,
		"DEEP_ARCHIVE":        tmtypes.StorageClassDeepArchive,
		"OUTPOSTS":            tmtypes.StorageClassOutposts,
		"GLACIER_IR":          tmtypes.StorageClassGlacierIr,
		"SNOW":                tmtypes.StorageClassSnow,
		"EXPRESS_ONEZONE":     tmtypes.StorageClassExpressOnezone,
		"NONSENSE":            tmtypes.StorageClassStandard,
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			assert.Equal(t, want, s.parseStorageClass(context.Background(), in))
		})
	}
}
