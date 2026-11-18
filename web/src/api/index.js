// All calls go through /api/...; Nginx (or the Vite dev proxy) routes them to the right service.

async function request(path, options, label) {
  const res = await fetch(path, options)
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || body.detail || `${label} error: ${res.status}`)
  }
  return res.json()
}

export const fetchMenu = () => request('/api/menu', undefined, 'menu-api')

export const fetchOrders = () => request('/api/orders', undefined, 'orders-api')

export const fetchStats = () => request('/api/stats', undefined, 'stats-api')

export const fetchInfo = () => request('/api/info', undefined, 'orders-api')

export const placeOrder = (customerName, items) =>
  request(
    '/api/orders',
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        customer_name: customerName,
        items: items.map(i => ({ menu_item_id: i.menu_item_id, quantity: i.quantity })),
      }),
    },
    'orders-api'
  )
