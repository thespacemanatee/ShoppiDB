import "antd/dist/antd.min.css"
import { useEffect } from "react"
import {
  Navigate,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from "react-router-dom"

import ContentLayout from "./components/ContentLayout"
import { useAppSelector } from "./features/hooks"
import Cart from "./routes/Cart"
import Home from "./routes/Home"
import Login from "./routes/Login"

function App() {
  const isLoggedIn = useAppSelector((state) => state.auth.isLoggedIn)
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    if (isLoggedIn) {
      if (location.pathname === "/login") {
        navigate("/home")
      }
    } else {
      navigate("/login")
    }
  }, [isLoggedIn, location.pathname, navigate])

  return (
    <Routes>
      {isLoggedIn ? (
        <Route path="/" element={<ContentLayout />}>
          <Route path="home" element={<Home />} />
          <Route path="cart" element={<Cart />} />
          <Route index element={<Navigate to="home" replace />} />
        </Route>
      ) : (
        <>
          <Route path="login" element={<Login />} />
          <Route index element={<Navigate to="login" replace />} />
        </>
      )}
    </Routes>
  )
}

export default App
