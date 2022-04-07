import { useEffect, useState } from "react"
import { Menu } from "antd"
import { useLocation, useNavigate } from "react-router-dom"

export default function TopNavigation() {
  const [currentKey, setCurrentKey] = useState("")

  const navigate = useNavigate()
  const location = useLocation()

  const handleClick = (info: any) => {
    navigate(info.key)
  }

  useEffect(() => {
    setCurrentKey(location.pathname)
  }, [location.pathname])

  return (
    <Menu onClick={handleClick} selectedKeys={[currentKey]} mode="horizontal">
      <Menu.Item key="/home">Home</Menu.Item>
      <Menu.Item key="/cart">Cart</Menu.Item>
      <Menu.Item key="/debug">Debug</Menu.Item>
    </Menu>
  )
}
