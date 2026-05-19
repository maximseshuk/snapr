import { memo } from 'react'

import { cn } from '@/lib/utils'

import type { ParsedLine, SearchMatch } from './parse'

interface LogRowProps {
  parsed: ParsedLine
  rowIndex: number
  lineNumber: number
  rowMatches: SearchMatch[]
  activeMatchIdx?: number
  style?: React.CSSProperties
  wrap?: boolean
}

interface LocalHL {
  start: number
  end: number
  active: boolean
}

const renderSegments = (parsed: ParsedLine, rowMatches: SearchMatch[], activeMatchIdx?: number): React.ReactNode[] => {
  const segments = parsed.segments
  if (rowMatches.length === 0) {
    const out: React.ReactNode[] = Array.from({ length: segments.length })
    for (let i = 0; i < segments.length; i++) {
      const s = segments[i]
      out[i] = (
        <span key={i} className={s.className}>
          {s.text}
        </span>
      )
    }
    return out
  }

  const out: React.ReactNode[] = []
  let cursor = 0
  let key = 0
  const local: LocalHL[] = []

  for (let si = 0; si < segments.length; si++) {
    const seg = segments[si]
    const segStart = cursor
    const segEnd = cursor + seg.text.length

    local.length = 0
    // rowMatches arrive in document order from findMatches → no sort needed.
    for (let mi = 0; mi < rowMatches.length; mi++) {
      const m = rowMatches[mi]
      if (m.end <= segStart) continue
      if (m.start >= segEnd) break
      local.push({
        start: Math.max(m.start, segStart) - segStart,
        end: Math.min(m.end, segEnd) - segStart,
        active: mi === activeMatchIdx,
      })
    }

    if (local.length === 0) {
      out.push(
        <span key={key++} className={seg.className}>
          {seg.text}
        </span>,
      )
      cursor = segEnd
      continue
    }

    let pos = 0
    for (let hi = 0; hi < local.length; hi++) {
      const h = local[hi]
      if (h.start > pos) {
        out.push(
          <span key={key++} className={seg.className}>
            {seg.text.slice(pos, h.start)}
          </span>,
        )
      }
      out.push(
        <mark
          key={key++}
          className={cn(
            seg.className,
            'rounded-sm',
            h.active ? 'bg-amber-400 text-black dark:bg-amber-300' : 'bg-amber-200/70 text-black dark:bg-amber-200/60',
          )}
        >
          {seg.text.slice(h.start, h.end)}
        </mark>,
      )
      pos = h.end
    }
    if (pos < seg.text.length) {
      out.push(
        <span key={key++} className={seg.className}>
          {seg.text.slice(pos)}
        </span>,
      )
    }
    cursor = segEnd
  }
  return out
}

export const LogRow = memo(function LogRow({
  parsed,
  rowIndex: _rowIndex,
  lineNumber,
  rowMatches,
  activeMatchIdx,
  style,
  wrap,
}: LogRowProps) {
  return (
    <div
      style={style}
      className={cn(
        'hover:bg-foreground/5 group flex items-start gap-3 px-3 leading-relaxed',
        wrap ? 'break-all whitespace-pre-wrap' : 'whitespace-pre',
      )}
    >
      <span className="log-number text-muted-foreground/60 border-border/60 w-12 shrink-0 border-r pr-2 text-right tabular-nums select-none">
        {lineNumber}
      </span>
      <span className="min-w-0 flex-1">{renderSegments(parsed, rowMatches, activeMatchIdx)}</span>
    </div>
  )
})
