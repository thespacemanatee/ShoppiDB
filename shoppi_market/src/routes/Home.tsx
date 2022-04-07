import { Breadcrumb, Button } from "antd"
import axios from "axios"

export default function VideoSurveillance() {
  const handleAddToCart = async () => {
    try {
      const res = await axios.post("http://localhost:8000/get", {
        key: "testkey",
        value: {},
      })
      console.log(res)
    } catch (err) {
      console.error(err)
    }
  }

  return (
    <div className="h-full min-h-screen">
      <Breadcrumb>
        <Breadcrumb.Item>Marketplace</Breadcrumb.Item>
      </Breadcrumb>
      <div className="mt-4 grid grid-cols-3 gap-4">
        {Array(5)
          .fill(0)
          .map((e, idx) => (
            <div
              key={idx}
              className="flex flex-col items-center justify-center"
            />
          ))}
      </div>
      <Button type="primary" shape="round" onClick={handleAddToCart}>
        Add to Cart
      </Button>
    </div>
  )
}
