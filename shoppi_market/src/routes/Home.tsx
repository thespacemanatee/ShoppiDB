import { Breadcrumb, Button } from "antd"
import { useState } from "react"

import { FoodItem } from "../features/cart/types"

import mockData from "../services/mockData"

export default function VideoSurveillance() {
  const [foods, setFoods] = useState<FoodItem[]>(mockData)

  const handleAddToCart = async () => {
    try {
      // const res = await getCartByKey()
    } catch (err) {
      console.error(err)
    }
  }

  console.log(foods)

  return (
    <div className="h-full min-h-screen">
      <Breadcrumb>
        <Breadcrumb.Item>Marketplace</Breadcrumb.Item>
      </Breadcrumb>
      <div className="mt-4 grid grid-cols-4 gap-8">
        {foods.map((e, idx) => (
          <div key={e.id}>
            <img
              src={e.imageUrl}
              alt={e.name}
              className="h-96 w-full overflow-hidden rounded-2xl object-cover shadow-lg"
            />
            <div className="mt-8">
              <div className="mb-2 text-xl">{e.name}</div>
              <div className="mb-2 text-xl">{`S$${e.price}`}</div>
              <Button type="primary" shape="round" onClick={handleAddToCart}>
                Add to Cart
              </Button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
