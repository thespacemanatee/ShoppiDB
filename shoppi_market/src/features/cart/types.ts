export type Item = {
  id: string
  name: string
  price: number
  quantity: number
}

export type ShoppingCart = {
  items: Item[]
}

export type Clock = {
  counter: number
  lastUpdated: number
}

export type Context = {
  [nodeId: string]: Clock
}
