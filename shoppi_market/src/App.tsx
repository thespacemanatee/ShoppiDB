import "antd/dist/antd.min.css"
import { Navigate, Route, Routes } from "react-router-dom"

import ContentLayout from "./components/ContentLayout"
import Home from "./routes/Home"

function App() {
  return (
    <Routes>
      <Route path="/" element={<ContentLayout />}>
        <Route path="home" element={<Home />} />
        <Route path="cart" element={undefined} />
        <Route index element={<Navigate to="home" replace />} />
      </Route>
    </Routes>
  )
}

export default App
