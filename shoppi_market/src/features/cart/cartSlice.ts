import { createSlice, PayloadAction } from "@reduxjs/toolkit"

import type { RootState } from "../store"
import { Context, FoodItem, Item } from "../types"

export interface CartState {
  key: string
  items: Item[]
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
    setCart: (state, action: PayloadAction<CartState>) => {
      state.key = action.payload.key
      state.items = action.payload.items
      state.context = action.payload.context
    },
    addItemToCart: (state, action: PayloadAction<FoodItem>) => {
      const updatedItem = state.items.find((e) => e.id === action.payload.id)
      if (updatedItem) {
        updatedItem.quantity += 1
      } else {
        state.items.push({
          id: action.payload.id,
          name: action.payload.name,
          price: action.payload.price,
          quantity: 1,
        })
      }
    },
    addOne: (state, action: PayloadAction<string>) => {
      const updatedItem = state.items.find((e) => e.id === action.payload)
      if (updatedItem) {
        updatedItem.quantity += 1
      }
    },
    removeOne: (state, action: PayloadAction<string>) => {
      const updatedItem = state.items.find((e) => e.id === action.payload)
      if (updatedItem) {
        if (updatedItem.quantity > 0) {
          updatedItem.quantity -= 1
        }
      }
    },
    removeItemFromCart: (state, action: PayloadAction<string>) => {
      state.items = state.items.filter((item) => item.id !== action.payload)
    },
  },
})

export const {
  setCartKey,
  setCart,
  addItemToCart,
  addOne,
  removeOne,
  removeItemFromCart,
} = cartSlice.actions
export const selectItems = (state: RootState) => state.cart.items
export default cartSlice.reducer
