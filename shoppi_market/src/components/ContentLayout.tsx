import { useEffect } from "react"
import { Layout } from "antd"
import { Outlet } from "react-router-dom"

import SidebarMenu from "./SidebarMenu"
import TopNavigation from "./TopNavigation"
import { getCartByKey } from "../services/api"
import { CART_KEY } from "../config/constants"

const { Content } = Layout

export default function ContentLayout() {
  useEffect(() => {
    ;(async () => {
      const res = await getCartByKey(CART_KEY)
      console.log(res)
    })()
  }, [])

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
