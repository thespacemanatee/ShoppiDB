import { Button } from "antd"
import { useNavigate } from "react-router-dom"

import { login } from "../features/auth/authSlice"
import { useAppDispatch } from "../features/hooks"

const Login = () => {
  const dispatch = useAppDispatch()
  const navigate = useNavigate()

  const handleLogin = () => {
    dispatch(login())
    navigate("/home")
  }

  return (
    <div className="flex h-screen flex-col items-center justify-center">
      <span className="mb-4 text-6xl font-semibold text-amber-500">
        ShoppiDB 🛍️
      </span>
      <div className="w-1/3">
        <Button type="primary" block onClick={handleLogin}>
          Login
        </Button>
      </div>
    </div>
  )
}

export default Login
