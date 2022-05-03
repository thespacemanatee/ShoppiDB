import { useEffect, useMemo, useState } from "react"
import { Menu, MenuProps } from "antd"
import { useLocation, useNavigate } from "react-router-dom"
import { useAppSelector } from "../features/cart/hooks"

export default function TopNavigation() {
  const cart = useAppSelector((state) => state.cart)
  const [currentKey, setCurrentKey] = useState("")

  const items: MenuProps["items"] = useMemo(
    () => [
      {
        label: "Home",
        key: "/home",
      },
      {
        label: `Cart (${cart.items.length})`,
        key: "/cart",
      },
    ],
    [cart.items.length]
  )

  const navigate = useNavigate()
  const location = useLocation()

  const handleClick = (info: any) => {
    navigate(info.key)
  }

  useEffect(() => {
    setCurrentKey(location.pathname)
  }, [location.pathname])

  return (
    <Menu
      onClick={handleClick}
      selectedKeys={[currentKey]}
      mode="horizontal"
      items={items}
    />
  )
}
