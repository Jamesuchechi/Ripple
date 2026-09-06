import React from 'react'

const navItems = [
  { id: 'metrics', label: 'Real-Time Metrics', icon: '📊' },
  { id: 'logs', label: 'Delivery Logs', icon: '⚡' },
  { id: 'workflows', label: 'Workflow Builder', icon: '🔀' },
  { id: 'templates', label: 'Template Editor', icon: '📝' },
  { id: 'dlq', label: 'DLQ Inspector', icon: '☠️' },
  { id: 'preferences', label: 'User Preferences', icon: '⚙️' },
  { id: 'apikeys', label: 'API Keys', icon: '🔑' },
  { id: 'usage', label: 'Usage & Billing', icon: '📈' },
]

export default function Sidebar({ activeView, setActiveView }) {
  return (
    <aside className="sidebar">
      <div className="logo-container">
        <div className="logo-icon">R</div>
        <div className="logo-text">Ripple</div>
      </div>
      <nav>
        <ul className="nav-list">
          {navItems.map((item) => (
            <li
              key={item.id}
              className={`nav-item ${activeView === item.id ? 'active' : ''}`}
              onClick={() => setActiveView(item.id)}
            >
              <span>{item.icon}</span>
              <span>{item.label}</span>
            </li>
          ))}
        </ul>
      </nav>
    </aside>
  )
}
