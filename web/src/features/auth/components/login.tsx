import { LoginForm } from './login-form'

interface LoginProps {
  onLoginSuccess: () => void
}

export const Login = ({ onLoginSuccess }: LoginProps) => {
  return (
    <div className="bg-muted flex min-h-screen items-center justify-center">
      <div className="w-full max-w-md p-6">
        <LoginForm onLoginSuccess={onLoginSuccess} />
      </div>
    </div>
  )
}
