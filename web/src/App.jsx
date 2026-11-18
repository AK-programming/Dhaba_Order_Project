import { useState } from 'react'
import OrderPage from './pages/OrderPage'
import KitchenScreen from './pages/KitchenScreen'
import StatsPanel from './pages/StatsPanel'

const TABS = [
  { id: 'order', label: 'Order' },
  { id: 'kitchen', label: 'Kitchen' },
  { id: 'stats', label: 'Stats' },
]

export default function App() {
  const [tab, setTab] = useState('order')

  return (
    <div className="app">
      <header className="app-header">
        <h1 className="app-title">Dhaba Orders</h1>
        <nav className="tab-nav">
          {TABS.map(t => (
            <button
              key={t.id}
              className={`tab-btn${tab === t.id ? ' active' : ''}`}
              onClick={() => setTab(t.id)}
            >
              {t.label}
            </button>
          ))}
        </nav>
      </header>

      <main className="app-main">
        {tab === 'order' && <OrderPage />}
        {tab === 'kitchen' && <KitchenScreen />}
        {tab === 'stats' && <StatsPanel />}
      </main>
    </div>
  )
}
