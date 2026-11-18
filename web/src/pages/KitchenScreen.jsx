import { useCallback, useEffect, useState } from 'react'
import { fetchOrders } from '../api'

// Status values match the orders.status column in Postgres.
const STATUS_ORDER = ['new', 'cooking', 'ready']
const STATUS_LABEL = { new: 'New', cooking: 'Cooking', ready: 'Ready' }

export default function KitchenScreen() {
  const [orders, setOrders] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  const load = useCallback(() => {
    fetchOrders()
      .then(data => {
        setOrders(data)
        setError(null)
      })
      .catch(e => setError(e.message))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    load()
    const id = setInterval(load, 5000)
    return () => clearInterval(id)
  }, [load])

  if (loading) return <div className="center-msg">Loading orders...</div>
  if (error) return <div className="center-msg error-msg">{error}</div>
  if (orders.length === 0) {
    return <div className="center-msg">No orders yet. Place one on the Order tab.</div>
  }

  const byStatus = STATUS_ORDER.reduce((acc, s) => {
    acc[s] = orders.filter(o => o.status === s)
    return acc
  }, {})

  return (
    <div className="kitchen-screen">
      <div className="kitchen-cols">
        {STATUS_ORDER.map(status => (
          <div key={status} className={`kitchen-col kitchen-col-${status}`}>
            <h2 className="kitchen-col-title">
              {STATUS_LABEL[status]}
              <span className="col-count">{byStatus[status].length}</span>
            </h2>
            <div className="kitchen-cards">
              {byStatus[status].length === 0 ? (
                <p className="empty-col">-</p>
              ) : (
                byStatus[status].map(order => (
                  <div key={order.id} className="kitchen-card">
                    <div className="kc-header">
                      <span className="kc-id">#{order.id}</span>
                      <span className="kc-name">{order.customer_name}</span>
                    </div>
                    <ul className="kc-items">
                      {order.items.map(item => (
                        <li key={`${order.id}-${item.menu_item_id}`}>
                          <span>{item.name}</span>
                          <span className="kc-qty">x{item.quantity}</span>
                        </li>
                      ))}
                    </ul>
                    <div className="kc-time">
                      {new Date(order.created_at).toLocaleTimeString([], {
                        hour: '2-digit',
                        minute: '2-digit',
                      })}
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        ))}
      </div>
      <p className="refresh-note">Refreshes every 5 seconds</p>
    </div>
  )
}
