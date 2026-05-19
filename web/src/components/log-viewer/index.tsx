import {
  IconChevronsDown,
  IconChevronsUp,
  IconDownload,
  IconMaximize,
  IconMinimize,
  IconTextWrap,
  IconTextWrapDisabled,
} from '@tabler/icons-react'
import { useVirtualizer } from '@tanstack/react-virtual'
import * as React from 'react'
import { useCallback, useEffect, useId, useLayoutEffect, useMemo, useRef, useState } from 'react'

import type { LogLine } from '@/types/api'

import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

import { LogRow } from './log-row'
import { LogSearchInput } from './log-search-input'
import { type ParsedLine, type SearchMatch, findMatches, parseLine } from './parse'

interface LogViewerProps extends React.HTMLAttributes<HTMLDivElement> {
  entries: LogLine[]
  resetKey?: string | number
  emptyMessage: string
  autoScrollLabel: string
  searchPlaceholder: string
  toolbar?: React.ReactNode
  estimatedRowHeight?: number
  minSearchChars?: number
  downloadFileName?: string
  fullscreenLabel?: string
  exitFullscreenLabel?: string
  downloadLabel?: string
  wrapLabel?: string
  unwrapLabel?: string
  jumpToTopLabel?: string
  jumpToBottomLabel?: string
}

const SEARCH_DEBOUNCE_MS = 150
const ANSI_RE = /\[[0-9;]*m/g

interface DisplayLine {
  line: string
  lineNumber: number
  isMarker?: boolean
}

const LogViewer = React.forwardRef<HTMLDivElement, LogViewerProps>(function LogViewer(
  {
    entries,
    resetKey,
    emptyMessage,
    autoScrollLabel,
    searchPlaceholder,
    toolbar,
    className,
    estimatedRowHeight = 18,
    minSearchChars = 2,
    downloadFileName = 'logs.txt',
    fullscreenLabel = 'Fullscreen',
    exitFullscreenLabel = 'Exit fullscreen',
    downloadLabel = 'Download',
    wrapLabel = 'Wrap text',
    unwrapLabel = 'Unwrap text',
    jumpToTopLabel = 'Jump to top',
    jumpToBottomLabel = 'Jump to bottom',
    ...props
  },
  ref,
) {
  const switchId = useId()
  const [followLogs, setFollowLogs] = useState(true)
  const [searchInput, setSearchInput] = useState('')
  const [searchQuery, setSearchQuery] = useState('')
  const [activeMatchIdx, setActiveMatchIdx] = useState(0)
  const [fullscreen, setFullscreen] = useState(false)
  const [wrap, setWrap] = useState(false)

  const resetKeyRef = useRef(resetKey)
  if (resetKeyRef.current !== resetKey) {
    resetKeyRef.current = resetKey
    if (searchInput) setSearchInput('')
    if (searchQuery) setSearchQuery('')
    if (activeMatchIdx !== 0) setActiveMatchIdx(0)
  }

  useEffect(() => {
    const t = setTimeout(() => {
      setSearchQuery(searchInput.length >= minSearchChars ? searchInput : '')
      setActiveMatchIdx(0)
    }, SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(t)
  }, [searchInput, minSearchChars])

  const displayLines = useMemo<DisplayLine[]>(() => {
    return Array.from({ length: entries.length }, (_, i) => ({
      line: entries[i],
      lineNumber: i + 1,
    }))
  }, [entries])

  const parseCacheRef = useRef<Map<string, ParsedLine>>(new Map())
  const parsedLines = useMemo<ParsedLine[]>(() => {
    const cache = parseCacheRef.current
    return displayLines.map(({ line }) => {
      const cached = cache.get(line)
      if (cached) return cached
      const parsed = parseLine(line)
      cache.set(line, parsed)
      return parsed
    })
  }, [displayLines])

  useEffect(() => {
    const cache = parseCacheRef.current
    if (cache.size > 5000) {
      const live = new Set(displayLines.map((d) => d.line))
      for (const k of cache.keys()) if (!live.has(k)) cache.delete(k)
    }
  }, [displayLines])

  const matches = useMemo<SearchMatch[]>(() => findMatches(parsedLines, searchQuery), [parsedLines, searchQuery])

  const matchesByRow = useMemo<Map<number, { match: SearchMatch; globalIdx: number }[]>>(() => {
    const map = new Map<number, { match: SearchMatch; globalIdx: number }[]>()
    matches.forEach((m, i) => {
      const arr = map.get(m.rowIndex) ?? []
      arr.push({ match: m, globalIdx: i })
      map.set(m.rowIndex, arr)
    })
    return map
  }, [matches])

  const scrollRef = useRef<HTMLDivElement>(null)
  const virtualizer = useVirtualizer({
    count: displayLines.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => estimatedRowHeight,
    overscan: 12,
    // Item key includes `wrap` so the virtualizer's measured-size cache is invalidated on toggle.
    getItemKey: (index) => `${wrap ? 'w' : 'nw'}-${index}`,
  })

  const programmaticScrollRef = useRef(0)
  const scrollToBottom = useCallback(() => {
    programmaticScrollRef.current = performance.now()
    virtualizer.scrollToIndex(displayLines.length - 1, { align: 'end' })
  }, [virtualizer, displayLines.length])

  const followLogsRef = useRef(followLogs)
  followLogsRef.current = followLogs
  useEffect(() => {
    if (followLogsRef.current && displayLines.length > 0) scrollToBottom()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [wrap, fullscreen, scrollToBottom])

  const lastCountRef = useRef(0)
  useLayoutEffect(() => {
    if (!followLogs) return
    if (displayLines.length === lastCountRef.current) return
    lastCountRef.current = displayLines.length
    if (displayLines.length > 0) scrollToBottom()
  }, [displayLines.length, followLogs, scrollToBottom])

  const lastSearchRef = useRef<{ q: string; idx: number }>({ q: '', idx: -1 })
  useEffect(() => {
    if (matches.length === 0) {
      lastSearchRef.current = { q: searchQuery, idx: -1 }
      return
    }
    if (lastSearchRef.current.q === searchQuery && lastSearchRef.current.idx === activeMatchIdx) return
    lastSearchRef.current = { q: searchQuery, idx: activeMatchIdx }
    const m = matches[activeMatchIdx]
    if (m) {
      programmaticScrollRef.current = performance.now()
      virtualizer.scrollToIndex(m.rowIndex, { align: 'center' })
    }
  }, [matches, activeMatchIdx, searchQuery, virtualizer])

  // Detect user scroll-away from bottom. Programmatic scrolls (scrollToBottom / search jump) set a
  // timestamp window during which scroll events are ignored. Avoids the wheel/touch heuristic which
  // missed scrollbar drags and fired before scrollTop updated.
  const scrollCleanupRef = useRef<(() => void) | null>(null)
  const setScrollEl = useCallback((el: HTMLDivElement | null) => {
    scrollCleanupRef.current?.()
    scrollCleanupRef.current = null
    scrollRef.current = el
    if (!el) return
    const onScroll = () => {
      if (performance.now() - programmaticScrollRef.current < 150) return
      const distance = el.scrollHeight - el.scrollTop - el.clientHeight
      if (distance > 4) {
        if (followLogsRef.current) setFollowLogs(false)
      } else if (!followLogsRef.current) {
        setFollowLogs(true)
      }
    }
    el.addEventListener('scroll', onScroll, { passive: true })
    scrollCleanupRef.current = () => {
      el.removeEventListener('scroll', onScroll)
    }
  }, [])
  useEffect(() => {
    return () => {
      scrollCleanupRef.current?.()
      scrollCleanupRef.current = null
    }
  }, [])

  const handlePrev = () => {
    if (matches.length === 0) return
    setFollowLogs(false)
    setActiveMatchIdx((i) => (i - 1 + matches.length) % matches.length)
  }
  const handleNext = () => {
    if (matches.length === 0) return
    setFollowLogs(false)
    setActiveMatchIdx((i) => (i + 1) % matches.length)
  }
  const handleClear = () => {
    setSearchInput('')
    setSearchQuery('')
    setActiveMatchIdx(0)
  }

  const handleJumpTop = () => {
    if (displayLines.length === 0) return
    setFollowLogs(false)
    virtualizer.scrollToIndex(0, { align: 'start' })
  }
  const handleJumpBottom = () => {
    if (displayLines.length === 0) return
    setFollowLogs(true)
    virtualizer.scrollToIndex(displayLines.length - 1, { align: 'end' })
  }

  const handleDownload = useCallback(() => {
    const text = entries.map((e) => e.replace(ANSI_RE, '')).join('\n')
    const blob = new Blob([text], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = downloadFileName
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }, [entries, downloadFileName])

  const rootRef = useRef<HTMLDivElement>(null)
  React.useImperativeHandle(ref, () => rootRef.current as HTMLDivElement)
  useEffect(() => {
    const onChange = () => setFullscreen(!!document.fullscreenElement)
    document.addEventListener('fullscreenchange', onChange)
    return () => document.removeEventListener('fullscreenchange', onChange)
  }, [])

  const toggleFullscreen = useCallback(() => {
    const el = rootRef.current
    if (!el) return
    if (document.fullscreenElement) {
      document.exitFullscreen?.()
    } else {
      el.requestFullscreen?.()
    }
  }, [])

  const virtualItems = virtualizer.getVirtualItems()

  return (
    <div
      ref={rootRef}
      className={cn(
        'flex w-full min-w-0 flex-col gap-3',
        fullscreen ? 'bg-background h-full p-4' : 'h-full',
        className,
      )}
      {...props}
    >
      <div className="flex flex-col gap-2 lg:flex-row lg:flex-wrap lg:items-center lg:justify-between">
        <div className="text-muted-foreground min-w-0 truncate text-sm">{toolbar}</div>
        <TooltipProvider delayDuration={300}>
          <div className="flex flex-wrap items-center gap-3">
            <LogSearchInput
              value={searchInput}
              onChange={setSearchInput}
              onClear={handleClear}
              onPrev={handlePrev}
              onNext={handleNext}
              total={matches.length}
              current={matches.length > 0 ? activeMatchIdx + 1 : 0}
              placeholder={searchPlaceholder}
              minSearchChars={minSearchChars}
            />
            <Separator orientation="vertical" className="hidden !h-6 !self-center sm:block" />
            <div className="flex items-center gap-1">
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    className="size-8"
                    onClick={() => setWrap((v) => !v)}
                    aria-label={wrap ? unwrapLabel : wrapLabel}
                  >
                    {wrap ? <IconTextWrapDisabled className="size-4" /> : <IconTextWrap className="size-4" />}
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{wrap ? unwrapLabel : wrapLabel}</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    className="size-8"
                    onClick={handleDownload}
                    disabled={entries.length === 0}
                    aria-label={downloadLabel}
                  >
                    <IconDownload className="size-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{downloadLabel}</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    className="size-8"
                    onClick={toggleFullscreen}
                    aria-label={fullscreen ? exitFullscreenLabel : fullscreenLabel}
                  >
                    {fullscreen ? <IconMinimize className="size-4" /> : <IconMaximize className="size-4" />}
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{fullscreen ? exitFullscreenLabel : fullscreenLabel}</TooltipContent>
              </Tooltip>
            </div>
            <Separator orientation="vertical" className="!h-6 !self-center" />
            <div className="flex items-center gap-1">
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    className="size-8"
                    onClick={handleJumpTop}
                    disabled={displayLines.length === 0}
                    aria-label={jumpToTopLabel}
                  >
                    <IconChevronsUp className="size-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{jumpToTopLabel}</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    className="size-8"
                    onClick={handleJumpBottom}
                    disabled={displayLines.length === 0}
                    aria-label={jumpToBottomLabel}
                  >
                    <IconChevronsDown className="size-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{jumpToBottomLabel}</TooltipContent>
              </Tooltip>
            </div>
            <Separator orientation="vertical" className="!h-6 !self-center" />
            <div className="flex items-center gap-2">
              <Label htmlFor={switchId} className="text-muted-foreground text-sm font-normal">
                {autoScrollLabel}
              </Label>
              <Switch id={switchId} checked={followLogs} onCheckedChange={setFollowLogs} />
            </div>
          </div>
        </TooltipProvider>
      </div>

      <div className="log-viewer relative min-h-0 flex-1 overflow-hidden rounded-md border font-mono text-xs">
        {displayLines.length === 0 ? (
          <div className="text-muted-foreground flex h-full items-center justify-center">{emptyMessage}</div>
        ) : (
          <div ref={setScrollEl} className="absolute inset-0 overflow-auto">
            <div style={{ height: virtualizer.getTotalSize(), width: '100%', position: 'relative' }}>
              {virtualItems.map((vi) => {
                const row = displayLines[vi.index]
                const rowMatches = matchesByRow.get(vi.index) ?? []
                const activeLocal = rowMatches.findIndex((rm) => rm.globalIdx === activeMatchIdx)
                const baseStyle = {
                  position: 'absolute' as const,
                  top: 0,
                  left: 0,
                  width: '100%',
                  transform: `translateY(${vi.start}px)`,
                }
                if (row.isMarker) {
                  return (
                    <div
                      key={vi.key}
                      ref={virtualizer.measureElement}
                      data-index={vi.index}
                      style={baseStyle}
                      className="text-muted-foreground border-b border-dashed px-3 py-1 text-xs italic"
                    >
                      {row.line}
                    </div>
                  )
                }
                return (
                  <div key={vi.key} ref={virtualizer.measureElement} data-index={vi.index} style={baseStyle}>
                    <LogRow
                      parsed={parsedLines[vi.index]}
                      rowIndex={vi.index}
                      lineNumber={row.lineNumber}
                      rowMatches={rowMatches.map((rm) => rm.match)}
                      activeMatchIdx={activeLocal >= 0 ? activeLocal : undefined}
                      wrap={wrap}
                    />
                  </div>
                )
              })}
            </div>
          </div>
        )}
      </div>
    </div>
  )
})
LogViewer.displayName = 'LogViewer'

export { LogViewer }
export type { LogViewerProps }
