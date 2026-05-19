const isServer = typeof document === 'undefined'
const DEFAULT_MAX_AGE = 60 * 60 * 24 * 7

export const getCookie = (name: string) => {
  if (isServer) return undefined
  return document.cookie
    .split('; ')
    .find((row) => row.startsWith(`${name}=`))
    ?.split('=')[1]
}

export const setCookie = (name: string, value: string, maxAge: number = DEFAULT_MAX_AGE) => {
  if (isServer) return
  document.cookie = `${name}=${value}; path=/; max-age=${maxAge}`
}

export const deleteCookie = (name: string) => {
  if (isServer) return
  document.cookie = `${name}=; path=/; max-age=0`
}
