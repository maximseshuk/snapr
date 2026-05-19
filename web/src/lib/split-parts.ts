const SUFFIX_LEN = 3

export const partSuffix = (index: number): string => {
  if (index < 0) return ''
  const out: string[] = Array.from({ length: SUFFIX_LEN })
  let n = index
  for (let i = SUFFIX_LEN - 1; i >= 0; i--) {
    out[i] = String.fromCharCode('a'.charCodeAt(0) + (n % 26))
    n = Math.floor(n / 26)
  }
  return out.join('')
}

export const partFilename = (baseId: string, index: number): string => `${baseId}.part-${partSuffix(index)}`

export const partFilenames = (baseId: string, count: number): string[] =>
  Array.from({ length: count }, (_, i) => partFilename(baseId, i))
