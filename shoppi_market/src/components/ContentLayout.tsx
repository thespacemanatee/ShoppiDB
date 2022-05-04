import { Layout } from "antd"
import { Outlet } from "react-router-dom"

import SidebarMenu from "./SidebarMenu"
import TopNavigation from "./TopNavigation"

const { Content } = Layout

export default function ContentLayout() {
  return (
    <Layout>
      <TopNavigation />
      <Layout>
        <SidebarMenu />
        <Layout className="px-6 py-4">
          <Content>
            <Outlet />
          </Content>
        </Layout>
      </Layout>
    </Layout>
  )
}
