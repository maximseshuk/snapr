import { useEffect, useReducer, useRef } from 'react'

import type { LogLine } from '@/types/api'

interface State {
  entries: LogLine[]
  // bumps on source switch so consumers can clear caches keyed on it
  resetToken: number
}

type Action = { type: 'replace'; entries: LogLine[] } | { type: 'append'; line: LogLine } | { type: 'reset' }

const reducer = (state: State, action: Action): State => {
  switch (action.type) {
    case 'replace':
      return { entries: action.entries, resetToken: state.resetToken + 1 }
    case 'append':
      return { entries: [...state.entries, action.line], resetToken: state.resetToken }
    case 'reset':
      return { entries: [], resetToken: state.resetToken + 1 }
  }
}

interface Options {
  streamUrl: string | null
  initialFetch: () => Promise<LogLine[] | null>
  // changing deps triggers full reset + reconnect; pass [] for always-on streams
  deps: ReadonlyArray<unknown>
  // caps in-memory buffer; older entries trimmed off the front
  maxEntries?: number
}

export const useLogStream = ({ streamUrl, initialFetch, deps, maxEntries }: Options) => {
  const [state, dispatch] = useReducer(reducer, { entries: [], resetToken: 0 })
  const entriesRef = useRef(state.entries)
  entriesRef.current = state.entries

  // eslint-disable-next-line
  useEffect(() => {
    let cancelled = false
    let es: EventSource | null = null

    dispatch({ type: 'reset' })

    if (!streamUrl) return

    void (async () => {
      try {
        const tail = await initialFetch()
        if (cancelled) return
        if (tail && tail.length > 0) {
          dispatch({ type: 'replace', entries: tail })
        }
      } catch {}

      if (cancelled) return

      es = new EventSource(streamUrl, { withCredentials: true })
      es.addEventListener('message', (ev) => {
        if (typeof ev.data !== 'string' || ev.data.length === 0) return
        let line: string
        try {
          const parsed = JSON.parse(ev.data) as { line?: string }
          if (typeof parsed.line !== 'string') return
          line = parsed.line
        } catch {
          line = ev.data
        }
        if (line === '') return
        const cap = maxEntries && maxEntries > 0 ? maxEntries : 0
        if (cap > 0 && entriesRef.current.length >= cap) {
          const trimmed = entriesRef.current.slice(entriesRef.current.length - cap + 1)
          dispatch({ type: 'replace', entries: [...trimmed, line] })
          return
        }
        dispatch({ type: 'append', line })
      })
      es.addEventListener('error', () => {})
    })()

    return () => {
      cancelled = true
      es?.close()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps)

  return state
}
