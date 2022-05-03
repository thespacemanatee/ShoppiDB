import { Button } from "antd"
import { motion } from "framer-motion"

import {
  addOne,
  removeItemFromCart,
  removeOne,
  setCart,
} from "../features/cart/cartSlice"
import { useAppDispatch, useAppSelector } from "../features/cart/hooks"
import { putCart } from "../services/api"

import mockData from "../services/mockData"

export default function Cart() {
  const cart = useAppSelector((state) => state.cart)

  const dispatch = useAppDispatch()

  const handleRemoveFromCart = async (foodId: string) => {
    try {
      dispatch(removeItemFromCart(foodId))
      const temp = cart.items.filter((e) => e.id !== foodId)
      const res = await putCart(cart.key, temp, cart.context)
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

  const handleAddOne = async (foodId: string) => {
    try {
      dispatch(addOne(foodId))
      const temp = [...cart.items].map((e) => ({ ...e }))
      const updatedItem = temp.find((e) => e.id === foodId)
      if (updatedItem) {
        updatedItem.quantity += 1
      } else {
        return
      }
      const res = await putCart(cart.key, temp, cart.context)
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

  const handleRemoveOne = async (foodId: string) => {
    try {
      dispatch(removeOne(foodId))
      const temp = [...cart.items].map((e) => ({ ...e }))
      const updatedItem = temp.find((e) => e.id === foodId)
      if (updatedItem) {
        if (updatedItem.quantity > 0) {
          updatedItem.quantity -= 1
        }
      } else {
        return
      }
      const res = await putCart(cart.key, temp, cart.context)
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
        {cart.items.map((food, idx) => (
          <motion.div key={food.id} layout>
            <img
              src={mockData.find((e) => e.id === food.id)?.imageUrl}
              alt={food.name}
              className="h-96 w-full overflow-hidden rounded-2xl object-cover shadow-lg"
            />
            <div className="mt-8">
              <div className="flex flex-row items-center justify-between">
                <div className="text-xl">{food.name}</div>
                <div className="text-xl">{`S$${food.price}`}</div>
              </div>
              <div className="my-4 flex flex-row items-center justify-center">
                <Button
                  shape="round"
                  className="mx-2"
                  onClick={() => {
                    handleRemoveOne(food.id)
                  }}
                >
                  -
                </Button>
                <div>{`Quantity: ${food.quantity}`}</div>
                <Button
                  shape="round"
                  className="mx-2"
                  onClick={() => {
                    handleAddOne(food.id)
                  }}
                >
                  +
                </Button>
              </div>
              <Button
                type="primary"
                shape="round"
                danger
                onClick={() => handleRemoveFromCart(food.id)}
              >
                Remove from Cart
              </Button>
            </div>
          </motion.div>
        ))}
      </div>
    </div>
  )
}
