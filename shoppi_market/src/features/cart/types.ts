export interface FoodItem {
  id: string
  name: string
  price: number
  imageUrl: string
}

export type Item = {
  id: string
  name: string
  price: number
  quantity: number
}

export type ShoppingCart = {
  items: Item[]
}

export type VectorClock = {
  counter: number
  lastUpdated: number
}

export type Context = {
  [nodeId: string]: VectorClock
}
