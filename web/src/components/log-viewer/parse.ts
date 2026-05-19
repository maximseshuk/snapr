import Anser from 'anser'

export interface AnsiSegment {
  text: string
  className: string
}

export interface ParsedLine {
  plain: string
  plainLower: string
  segments: AnsiSegment[]
}

const colorMap: Record<string, string> = {
  'ansi-black': 'black',
  'ansi-red': 'red',
  'ansi-green': 'green',
  'ansi-yellow': 'yellow',
  'ansi-blue': 'blue',
  'ansi-magenta': 'magenta',
  'ansi-cyan': 'cyan',
  'ansi-white': 'white',
  'ansi-bright-black': 'grey',
  'ansi-bright-red': 'red',
  'ansi-bright-green': 'green',
  'ansi-bright-yellow': 'yellow',
  'ansi-bright-blue': 'blue',
  'ansi-bright-magenta': 'magenta',
  'ansi-bright-cyan': 'cyan',
  'ansi-bright-white': 'white',
}

// Output format `log-part {color}[Bold]` is consumed by CSS attribute selectors elsewhere — do not rename.
const classFromAnser = (fg: string | null, decorations: string[]): string => {
  if (!fg) return 'log-part'
  const base = colorMap[fg]
  if (!base) return 'log-part'
  const bold = decorations.includes('bold') || fg.startsWith('ansi-bright-')
  return `log-part ${base}${bold ? 'Bold' : ''}`
}

const PLAIN_SEG_CLASS = 'log-part'

export const parseLine = (line: string): ParsedLine => {
  // Fast path: no ESC byte → no ANSI to parse, skip Anser entirely.
  if (line.indexOf('\x1b') === -1) {
    return { plain: line, plainLower: line.toLowerCase(), segments: [{ text: line, className: PLAIN_SEG_CLASS }] }
  }
  const chunks = Anser.ansiToJson(line, { use_classes: true, json: true })
  const segments: AnsiSegment[] = []
  let plain = ''
  for (let i = 0; i < chunks.length; i++) {
    const c = chunks[i]
    if (!c.content) continue
    const decorations: string[] = (c.decorations as unknown as string[] | undefined) ?? []
    segments.push({ text: c.content, className: classFromAnser(c.fg as string | null, decorations) })
    plain += c.content
  }
  if (segments.length === 0) {
    return { plain: line, plainLower: line.toLowerCase(), segments: [{ text: line, className: PLAIN_SEG_CLASS }] }
  }
  return { plain, plainLower: plain.toLowerCase(), segments }
}

export interface SearchMatch {
  rowIndex: number
  start: number
  end: number
}

export const findMatches = (parsed: ParsedLine[], query: string, caseInsensitive = true): SearchMatch[] => {
  if (!query) return []
  const matches: SearchMatch[] = []
  const needle = caseInsensitive ? query.toLowerCase() : query
  const needleLen = needle.length
  for (let i = 0; i < parsed.length; i++) {
    const hay = caseInsensitive ? parsed[i].plainLower : parsed[i].plain
    let from = 0
    for (;;) {
      const idx = hay.indexOf(needle, from)
      if (idx < 0) break
      matches.push({ rowIndex: i, start: idx, end: idx + needleLen })
      from = idx + needleLen
    }
  }
  return matches
}
