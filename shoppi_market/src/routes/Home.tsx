import { Breadcrumb, Button } from "antd"
import { nanoid } from "nanoid"

import { addItemToCart, setCart } from "../features/cart/cartSlice"
import { useAppDispatch, useAppSelector } from "../features/cart/hooks"
import { FoodItem } from "../features/cart/types"
import { putCart } from "../services/api"

import mockData from "../services/mockData"

export default function VideoSurveillance() {
  const cart = useAppSelector((state) => state.cart)

  const dispatch = useAppDispatch()

  const handleAddToCart = async (item: FoodItem) => {
    try {
      let key = cart.key
      if (!key) {
        key = nanoid()
      }
      dispatch(addItemToCart(item))
      const temp = [...cart.items].map((e) => ({ ...e }))
      const updatedItem = temp.find((e) => e.id === item.id)
      if (updatedItem) {
        updatedItem.quantity += 1
      } else {
        temp.push({
          id: item.id,
          name: item.name,
          price: item.price,
          quantity: 1,
        })
      }
      const res = await putCart(key, temp, cart.context)
      const newCart = res?.data
      const items = JSON.parse(newCart?.value)
      const context = newCart.context
      console.log(newCart.key, items, context)
      dispatch(
        setCart({
          key: newCart.key,
          items,
          context,
        })
      )
    } catch (err) {
      console.error(err)
    }
  }

  return (
    <div className="h-full min-h-screen">
      <Breadcrumb>
        <Breadcrumb.Item>Marketplace</Breadcrumb.Item>
      </Breadcrumb>
      <div className="mt-4 grid grid-cols-4 gap-8">
        {mockData.map((food, idx) => (
          <div key={food.id}>
            <img
              src={food.imageUrl}
              alt={food.name}
              className="h-96 w-full overflow-hidden rounded-2xl object-cover shadow-lg"
            />
            <div className="mt-8">
              <div className="mb-2 text-xl">{food.name}</div>
              <div className="mb-2 text-xl">{`S$${food.price}`}</div>
              <Button
                type="primary"
                shape="round"
                onClick={() => {
                  handleAddToCart(food)
                }}
              >
                Add to Cart
              </Button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
