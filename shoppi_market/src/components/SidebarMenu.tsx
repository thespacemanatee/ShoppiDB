import { useMemo } from "react"
import { Button, Layout, Menu, MenuProps } from "antd"
import { UserOutlined } from "@ant-design/icons"

import { useAppDispatch } from "../features/hooks"
import { logout } from "../features/auth/authSlice"
import { persistor } from "../features/store"

const { Sider } = Layout

export default function SidebarMenu() {
  const dispatch = useAppDispatch()

  const handleLogout = async () => {
    dispatch(logout())
    await persistor.purge()
  }

  const items: MenuProps["items"] = useMemo(
    () => [
      {
        label: "Categories",
        icon: <UserOutlined />,
        key: "/categories",
        children: [
          {
            type: "group",
            label: "Main Course",
            key: "main-course",
            children: [
              {
                label: "Breakfast",
                key: "/breakfast",
              },
              {
                label: "Lunch",
                key: "/lunch",
              },
            ],
          },
          {
            type: "group",
            label: "Dried Foods",
            key: "dried-foods",
            children: [
              {
                label: "Snacks",
                key: "/snacks",
              },
              {
                label: "Biscuits",
                key: "/biscuits",
              },
            ],
          },
        ],
      },
    ],
    []
  )

  return (
    <Sider>
      <Menu
        className="h-full"
        mode="inline"
        defaultSelectedKeys={["/lunch"]}
        defaultOpenKeys={["/categories"]}
        items={items}
      />
      <Button type="primary" block onClick={handleLogout}>
        Logout
      </Button>
    </Sider>
  )
}
