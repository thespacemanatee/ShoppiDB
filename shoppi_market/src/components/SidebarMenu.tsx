import { Layout, Menu } from "antd"
import { UserOutlined } from "@ant-design/icons"

const { SubMenu } = Menu
const { Sider } = Layout

export default function SidebarMenu() {
  return (
    <Sider>
      <Menu
        className="h-full"
        mode="inline"
        defaultSelectedKeys={["1"]}
        defaultOpenKeys={["sub1"]}
      >
        <SubMenu key="sub1" icon={<UserOutlined />} title="subnav 1">
          <Menu.Item key="1">option1</Menu.Item>
          <Menu.Item key="2">option2</Menu.Item>
          <Menu.Item key="3">option3</Menu.Item>
          <Menu.Item key="4">option4</Menu.Item>
        </SubMenu>
      </Menu>
    </Sider>
  )
}
