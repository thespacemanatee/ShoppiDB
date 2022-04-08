import { createSlice, PayloadAction } from "@reduxjs/toolkit"

import type { RootState } from "./store"
import { Context, FoodItem } from "./types"

interface CartState {
  key: string
  items: FoodItem[]
  context: Context
}

const initialState: CartState = {
  key: "",
  items: [],
  context: {},
}

export const cartSlice = createSlice({
  name: "cart",
  initialState,
  reducers: {
    setCartKey: (state, action: PayloadAction<string>) => {
      state.key = action.payload
    },
    addItemToCart: (state, action: PayloadAction<FoodItem>) => {
      state.items.push(action.payload)
    },
  },
})

export const { setCartKey, addItemToCart } = cartSlice.actions
export const selectItems = (state: RootState) => state.cart.items
export default cartSlice.reducer
