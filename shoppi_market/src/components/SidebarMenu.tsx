import { Layout, Menu, MenuProps } from "antd"
import { UserOutlined } from "@ant-design/icons"
import { useMemo } from "react"

const { Sider } = Layout

export default function SidebarMenu() {
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
    </Sider>
  )
}
