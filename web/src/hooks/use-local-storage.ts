import { useState } from 'react'

import { getItem, setItem } from '@/lib/storage'

export const useLocalStorage = <T>(key: string, initialValue: T) => {
  const [storedValue, setStoredValue] = useState<T>(() => {
    const item = getItem(key)
    if (item === undefined) return initialValue
    try {
      return JSON.parse(item) as T
    } catch (error) {
      console.error(error)
      return initialValue
    }
  })

  const setValue = (value: T | ((val: T) => T)) => {
    const valueToStore = typeof value === 'function' ? (value as (val: T) => T)(storedValue) : value
    setStoredValue(valueToStore)
    setItem(key, JSON.stringify(valueToStore))
  }

  return [storedValue, setValue] as const
}
