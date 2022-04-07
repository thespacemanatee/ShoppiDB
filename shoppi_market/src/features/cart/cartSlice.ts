import { createSlice, PayloadAction } from "@reduxjs/toolkit"

import type { RootState } from "./store"
import { FoodItem } from "./types"

interface CartState {
  items: FoodItem[]
}

const initialState: CartState = {
  items: [],
}

export const cartSlice = createSlice({
  name: "cart",
  initialState,
  reducers: {
    addItemToCart: (state, action: PayloadAction<FoodItem>) => {
      state.items.push(action.payload)
    },
  },
})

export const { addItemToCart } = cartSlice.actions
export const selectItems = (state: RootState) => state.cart.items
export default cartSlice.reducer
