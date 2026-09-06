import React, { useState } from 'react'

const initialKeys = [
  {
    id: 'key_01',
    name: 'Production Ingestion Key',
    prefix: 'rip_live_9a8f...',
    fullKey: 'rip_live_9a8f23b10c4d7e6f8a9b0c1d2e3f4a5b',
    scope: 'Full Access',
    createdAt: '2026-09-01',
    revoked: false,
  },
  {
    id: 'key_02',
    name: 'Staging Integration Key',
    prefix: 'rip_test_1b2c...',
    fullKey: 'rip_test_1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e',
    scope: 'Ingest Only',
    createdAt: '2026-09-04',
    revoked: false,
  },
]

export default function APIKeysView() {
  const [keys, setKeys] = useState(initialKeys)
  const [newKeyName, setNewKeyName] = useState('')
  const [copiedKeyID, setCopiedKeyID] = useState(null)

  const handleCreateKey = (e) => {
    e.preventDefault()
    if (!newKeyName) return
    const randomHex = Array.from({ length: 24 }, () =>
      Math.floor(Math.random() * 16).toString(16)
    ).join('')
    const newKey = {
      id: 'key_' + Math.random().toString(36).substring(2, 7),
      name: newKeyName,
      prefix: `rip_live_${randomHex.substring(0, 4)}...`,
      fullKey: `rip_live_${randomHex}`,
      scope: 'Full Access',
      createdAt: new Date().toISOString().split('T')[0],
      revoked: false,
    }
    setKeys([newKey, ...keys])
    setNewKeyName('')
  }

  const handleRevokeKey = (id) => {
    setKeys(
      keys.map((k) => {
        if (k.id === id) return { ...k, revoked: true }
        return k
      })
    )
  }

  const handleCopy = (keyItem) => {
    navigator.clipboard.writeText(keyItem.fullKey)
    setCopiedKeyID(keyItem.id)
    setTimeout(() => setCopiedKeyID(null), 2000)
  }

  return (
    <div>
      <div className="section-header">
        <div>
          <h1 className="section-title">API Keys & Project Credentials</h1>
          <p className="section-desc">
            Manage tenant API keys, SHA-256 Bearer tokens, and multi-tenant security credentials.
          </p>
        </div>
      </div>

      <div className="glass-card" style={{ marginBottom: '1.5rem' }}>
        <h3 style={{ marginBottom: '1rem', fontSize: '1rem' }}>Create New API Key</h3>
        <form onSubmit={handleCreateKey} style={{ display: 'flex', gap: '1rem' }}>
          <input
            className="input"
            placeholder="Key Description / Service Name (e.g., Billing Microservice)"
            value={newKeyName}
            onChange={(e) => setNewKeyName(e.target.value)}
            style={{ flex: 1 }}
            required
          />
          <button type="submit" className="btn btn-primary">
            + Generate Secret Key
          </button>
        </form>
      </div>

      <div className="glass-card" style={{ padding: 0, overflow: 'hidden' }}>
        <table className="table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Key Token</th>
              <th>Scope</th>
              <th>Created</th>
              <th>Status</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            {keys.map((k) => (
              <tr key={k.id}>
                <td style={{ fontWeight: 600 }}>{k.name}</td>
                <td className="code-font" style={{ color: 'var(--accent-cyan)' }}>
                  {k.prefix}
                </td>
                <td>
                  <span className="badge badge-purple">{k.scope}</span>
                </td>
                <td style={{ color: 'var(--text-muted)' }}>{k.createdAt}</td>
                <td>
                  <span className={`badge ${k.revoked ? 'badge-error' : 'badge-success'}`}>
                    {k.revoked ? 'Revoked' : 'Active'}
                  </span>
                </td>
                <td>
                  <div style={{ display: 'flex', gap: '0.5rem' }}>
                    <button
                      className="btn btn-secondary"
                      style={{ padding: '0.25rem 0.6rem', fontSize: '0.75rem' }}
                      onClick={() => handleCopy(k)}
                      disabled={k.revoked}
                    >
                      {copiedKeyID === k.id ? 'Copied!' : 'Copy Secret'}
                    </button>
                    {!k.revoked && (
                      <button
                        className="btn btn-danger"
                        style={{ padding: '0.25rem 0.6rem', fontSize: '0.75rem' }}
                        onClick={() => handleRevokeKey(k.id)}
                      >
                        Revoke
                      </button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
