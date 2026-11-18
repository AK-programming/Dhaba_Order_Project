import { useEffect, useState } from 'react'
import { fetchMenu, placeOrder } from '../api'

export default function OrderPage() {
  const [menu, setMenu] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [cart, setCart] = useState({}) // { menuItemId: quantity }
  const [customerName, setCustomerName] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [submitted, setSubmitted] = useState(null)

  useEffect(() => {
    fetchMenu()
      .then(setMenu)
      .catch(e => setError(e.message))
      .finally(() => setLoading(false))
  }, [])

  function addToCart(item) {
    setCart(prev => ({ ...prev, [item.id]: (prev[item.id] || 0) + 1 }))
  }

  function removeFromCart(itemId) {
    setCart(prev => {
      const next = { ...prev }
      if (next[itemId] > 1) next[itemId] -= 1
      else delete next[itemId]
      return next
    })
  }

  const cartItems = menu
    .filter(m => cart[m.id])
    .map(m => ({ menu_item_id: m.id, name: m.name, price: m.price, quantity: cart[m.id] }))

  const cartTotal = cartItems.reduce((sum, i) => sum + i.price * i.quantity, 0)

  async function handleSubmit(e) {
    e.preventDefault()
    if (!customerName.trim() || cartItems.length === 0) return
    setSubmitting(true)
    setError(null)
    try {
      const result = await placeOrder(customerName.trim(), cartItems)
      setSubmitted(result)
      setCart({})
      setCustomerName('')
    } catch (err) {
      setError(err.message)
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) return <div className="center-msg">Loading menu...</div>

  const categories = [...new Set(menu.map(m => m.category))]

  return (
    <div className="order-page">
      {submitted && (
        <div className="banner banner-success">
          Order #{submitted.id} placed. Watch the Kitchen tab for updates.
          <button className="banner-close" onClick={() => setSubmitted(null)}>x</button>
        </div>
      )}
      {error && (
        <div className="banner banner-error">
          {error}
          <button className="banner-close" onClick={() => setError(null)}>x</button>
        </div>
      )}

      <div className="order-layout">
        <section className="menu-section">
          <h2>Menu</h2>
          {categories.map(cat => (
            <div key={cat} className="menu-category">
              <h3 className="category-label">{cat}</h3>
              <div className="menu-grid">
                {menu.filter(m => m.category === cat).map(item => (
                  <div key={item.id} className="menu-card">
                    <div className="menu-card-name">{item.name}</div>
                    <div className="menu-card-desc">{item.description}</div>
                    <div className="menu-card-footer">
                      <span className="menu-price">Rs {item.price}</span>
                      <button className="btn btn-add" onClick={() => addToCart(item)}>
                        Add
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </section>

        <aside className="cart-section">
          <h2>Your order</h2>
          {cartItems.length === 0 ? (
            <p className="empty-col">Nothing added yet.</p>
          ) : (
            <ul className="cart-list">
              {cartItems.map(i => (
                <li key={i.menu_item_id} className="cart-row">
                  <span className="cart-name">{i.name}</span>
                  <span className="cart-controls">
                    <button className="btn btn-qty" onClick={() => removeFromCart(i.menu_item_id)}>-</button>
                    <span className="cart-qty">{i.quantity}</span>
                    <button
                      className="btn btn-qty"
                      onClick={() => addToCart({ id: i.menu_item_id })}
                    >
                      +
                    </button>
                  </span>
                  <span className="cart-line">Rs {i.price * i.quantity}</span>
                </li>
              ))}
            </ul>
          )}

          <div className="cart-total">
            <span>Total</span>
            <strong>Rs {cartTotal}</strong>
          </div>

          <form onSubmit={handleSubmit} className="cart-form">
            <label htmlFor="customer-name">Name for the order</label>
            <input
              id="customer-name"
              type="text"
              value={customerName}
              onChange={e => setCustomerName(e.target.value)}
              placeholder="e.g. Ali"
              maxLength={100}
            />
            <button
              type="submit"
              className="btn btn-primary"
              disabled={submitting || !customerName.trim() || cartItems.length === 0}
            >
              {submitting ? 'Placing...' : 'Place order'}
            </button>
          </form>
        </aside>
      </div>
    </div>
  )
}
