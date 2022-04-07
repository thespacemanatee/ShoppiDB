import axios from "axios"
import { nanoid } from "nanoid"

import { Context, ShoppingCart } from "../features/cart/types"

export const getCartByKey = async (key: string) => {
  try {
    return await axios.post("http://localhost:8000/get", {
      key,
    })
  } catch (err) {
    console.error(err)
  }
}

export const putCart = async (value: ShoppingCart, context?: Context) => {
  try {
    return await axios.post("http://localhost:8000/put", {
      key: nanoid(),
      value,
      context,
    })
  } catch (err) {
    console.error(err)
  }
}
