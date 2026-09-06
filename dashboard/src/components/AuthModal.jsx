import React, { useState } from 'react'

export default function AuthModal({ isOpen, onClose, user, setUser }) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  if (!isOpen) return null

  const handleLogin = (e) => {
    e.preventDefault()
    if (!email) return
    setUser({ email, id: 'usr_' + Math.random().toString(36).substring(2, 9) })
    onClose()
  }

  const handleLogout = () => {
    setUser(null)
    onClose()
  }

  return (
    <div
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        backgroundColor: 'rgba(0,0,0,0.7)',
        backdropFilter: 'blur(8px)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 100,
      }}
    >
      <div className="glass-card" style={{ width: '380px' }}>
        <h2 style={{ marginBottom: '1rem', fontSize: '1.25rem' }}>
          {user ? 'Account Settings' : 'Sign in to Ripple'}
        </h2>

        {user ? (
          <div>
            <p style={{ color: 'var(--text-secondary)', marginBottom: '1.5rem' }}>
              Signed in as <strong>{user.email}</strong>
            </p>
            <button
              className="btn btn-danger"
              style={{ width: '100%' }}
              onClick={handleLogout}
            >
              Sign Out
            </button>
          </div>
        ) : (
          <form onSubmit={handleLogin}>
            <div style={{ marginBottom: '1rem' }}>
              <label
                style={{
                  display: 'block',
                  fontSize: '0.8rem',
                  color: 'var(--text-muted)',
                  marginBottom: '0.35rem',
                }}
              >
                Work Email
              </label>
              <input
                className="input"
                type="email"
                placeholder="dev@company.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
            </div>
            <div style={{ marginBottom: '1.5rem' }}>
              <label
                style={{
                  display: 'block',
                  fontSize: '0.8rem',
                  color: 'var(--text-muted)',
                  marginBottom: '0.35rem',
                }}
              >
                Password
              </label>
              <input
                className="input"
                type="password"
                placeholder="••••••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>
            <div style={{ display: 'flex', gap: '0.5rem' }}>
              <button
                type="button"
                className="btn btn-secondary"
                style={{ flex: 1 }}
                onClick={onClose}
              >
                Cancel
              </button>
              <button type="submit" className="btn btn-primary" style={{ flex: 1 }}>
                Continue
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}
