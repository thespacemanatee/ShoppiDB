import { Button } from "antd"

import { addItemToCart, setCart } from "../features/cart/cartSlice"
import { useAppDispatch, useAppSelector } from "../features/hooks"
import { FoodItem } from "../features/types"
import { getCartByKey, putCart } from "../services/api"
import { CART_KEY } from "../config/constants"

import mockData from "../services/mockData"
import { useEffect } from "react"

export default function Home() {
  const cart = useAppSelector((state) => state.cart)

  const dispatch = useAppDispatch()

  useEffect(() => {
    ;(async () => {
      const res = await getCartByKey(CART_KEY)
      console.log(res)
    })()
  }, [])

  const handleAddToCart = async (item: FoodItem) => {
    try {
      let key = cart.key
      if (!key) {
        key = CART_KEY
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
