import { IconLock, IconUser } from '@tabler/icons-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Logo } from '@/components/logo'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Field, FieldGroup } from '@/components/ui/field'
import { InputGroup, InputGroupAddon, InputGroupInput } from '@/components/ui/input-group'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { useSystem } from '@/hooks/use-system'
import { apiClient } from '@/lib/api'
import { cn } from '@/lib/utils'

export const LoginForm = ({
  className,
  onLoginSuccess,
  ...props
}: React.ComponentProps<'div'> & {
  onLoginSuccess: () => void
}) => {
  const { t } = useTranslation()
  const { data: systemData, isLoading: versionLoading } = useSystem({ staleTime: 60_000 })
  const version = systemData?.version
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)

    try {
      await apiClient.login(username, password)
      toast.success(t('success.login'))
      onLoginSuccess()
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : t('login.error')
      toast.error(errorMessage)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className={cn('flex flex-col gap-6', className)} {...props}>
      <div className="flex flex-col items-center gap-2 text-center">
        <a href="#" className="flex flex-col items-center gap-2 font-medium">
          <div className="flex size-12 items-center justify-center rounded-md">
            <Logo size={48} />
          </div>
          <span className="sr-only">snapr</span>
        </a>
        <h1 className="text-xl font-bold">{t('login.title')}</h1>
      </div>
      <Card>
        <CardContent>
          <form onSubmit={handleSubmit}>
            <FieldGroup>
              <Field>
                <InputGroup>
                  <InputGroupAddon>
                    <IconUser />
                  </InputGroupAddon>
                  <InputGroupInput
                    id="username"
                    type="text"
                    placeholder={t('login.username')}
                    required
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    disabled={isLoading}
                  />
                </InputGroup>
              </Field>
              <Field>
                <InputGroup>
                  <InputGroupAddon>
                    <IconLock />
                  </InputGroupAddon>
                  <InputGroupInput
                    id="password"
                    type="password"
                    placeholder={t('login.password')}
                    required
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    disabled={isLoading}
                  />
                </InputGroup>
              </Field>
              <Button type="submit" className="w-full" disabled={isLoading}>
                {isLoading ? <Spinner className="size-4" /> : t('login.submit')}
              </Button>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>
      <div className="flex justify-center">
        {versionLoading || !version ? (
          <Skeleton className="h-4 w-16" />
        ) : (
          <div className="text-muted-foreground text-xs">{version}</div>
        )}
      </div>
    </div>
  )
}
