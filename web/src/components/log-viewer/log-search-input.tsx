import { IconChevronDown, IconChevronUp, IconSearch, IconX } from '@tabler/icons-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

interface LogSearchInputProps {
  value: string
  onChange: (v: string) => void
  onClear: () => void
  onPrev: () => void
  onNext: () => void
  total: number
  current: number
  placeholder: string
  minSearchChars: number
  className?: string
}

export const LogSearchInput = ({
  value,
  onChange,
  onClear,
  onPrev,
  onNext,
  total,
  current,
  placeholder,
  minSearchChars,
  className,
}: LogSearchInputProps) => {
  const hasMatches = total > 0
  const noMatches = value.length >= minSearchChars && total === 0

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      if (hasMatches) (e.shiftKey ? onPrev : onNext)()
    } else if (e.key === 'Escape') {
      onClear()
    }
  }

  let counter = ''
  if (hasMatches) counter = `${current}/${total}`
  else if (noMatches) counter = '0/0'

  return (
    <div className={cn('flex w-full items-center gap-1.5 sm:w-auto', className)}>
      <div className="relative w-full sm:w-auto">
        <IconSearch className="text-muted-foreground pointer-events-none absolute top-1/2 left-2 size-4 -translate-y-1/2" />
        <Input
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          className="h-8 w-full pr-16 pl-7 text-sm sm:w-56"
        />
        <div className="text-muted-foreground pointer-events-none absolute top-1/2 right-2 -translate-y-1/2 text-xs tabular-nums">
          {counter}
        </div>
      </div>
      {value.length > 0 && (
        <>
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="size-8"
            onClick={onPrev}
            disabled={!hasMatches}
            aria-label="Previous match"
          >
            <IconChevronUp className="size-4" />
          </Button>
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="size-8"
            onClick={onNext}
            disabled={!hasMatches}
            aria-label="Next match"
          >
            <IconChevronDown className="size-4" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-8"
            onClick={onClear}
            aria-label="Clear search"
          >
            <IconX className="size-4" />
          </Button>
        </>
      )}
    </div>
  )
}
