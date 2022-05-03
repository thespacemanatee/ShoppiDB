import axios from "axios"

import { Context, ShoppingCart } from "../features/types"

export const getCartByKey = async (key: string) => {
  try {
    return await axios.post("http://localhost:8000/get", {
      key,
    })
  } catch (err) {
    console.error(err)
  }
}

export const putCart = async (
  key: string,
  value: ShoppingCart,
  context?: Context
) => {
  try {
    return await axios.post("http://localhost:8000/put", {
      key,
      value: JSON.stringify(value),
      context,
    })
  } catch (err) {
    console.error(err)
  }
}
