import React, { useState, useEffect } from 'react'

export default function APIKeysView() {
  const [keys, setKeys] = useState([])
  const [loading, setLoading] = useState(true)
  const [newKeyName, setNewKeyName] = useState('')
  const [copiedKeyID, setCopiedKeyID] = useState(null)
  const [createdSecret, setCreatedSecret] = useState(null)

  const fetchKeys = async () => {
    try {
      setLoading(true)
      const res = await fetch('/v1/admin/api-keys', {
        headers: { 'X-Project-ID': '00000000-0000-0000-0000-000000000001' },
      })
      if (res.ok) {
        const data = await res.json()
        setKeys(
          data.map((k) => ({
            id: k.id,
            name: k.name,
            prefix: k.raw_api_key ? `${k.raw_api_key.substring(0, 12)}...` : (k.key_hash ? `${k.key_hash.substring(0, 8)}...` : 'rip_live_...'),
            fullKey: k.raw_api_key || k.key_hash,
            scope: 'Full Access',
            createdAt: k.created_at ? k.created_at.split('T')[0] : new Date().toISOString().split('T')[0],
            revoked: !!k.revoked_at,
          }))
        )
      }
    } catch (err) {
      console.error('Failed fetching API keys:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchKeys()
  }, [])

  const handleCreateKey = async (e) => {
    e.preventDefault()
    if (!newKeyName) return
    try {
      const res = await fetch('/v1/admin/api-keys', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Project-ID': '00000000-0000-0000-0000-000000000001',
        },
        body: JSON.stringify({ name: newKeyName }),
      })
      if (res.ok) {
        const data = await res.json()
        setCreatedSecret(data.raw_api_key)
        setNewKeyName('')
        fetchKeys()
      }
    } catch (err) {
      console.error('Failed creating API key:', err)
    }
  }

  const handleRevokeKey = async (id) => {
    try {
      const res = await fetch(`/v1/admin/api-keys/${id}`, {
        method: 'DELETE',
        headers: { 'X-Project-ID': '00000000-0000-0000-0000-000000000001' },
      })
      if (res.ok) {
        fetchKeys()
      }
    } catch (err) {
      console.error('Failed revoking API key:', err)
    }
  }

  const handleCopy = (secretStr, id) => {
    navigator.clipboard.writeText(secretStr)
    setCopiedKeyID(id)
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

      {createdSecret && (
        <div className="glass-card" style={{ marginBottom: '1.5rem', borderColor: 'var(--accent-emerald)' }}>
          <h4 style={{ color: 'var(--accent-emerald)', marginBottom: '0.5rem' }}>
            New Secret Key Generated! Copy it now (it won't be displayed again):
          </h4>
          <div style={{ display: 'flex', gap: '1rem', alignItems: 'center' }}>
            <code className="code-font" style={{ flex: 1, padding: '0.5rem', background: 'rgba(0,0,0,0.4)', borderRadius: '4px' }}>
              {createdSecret}
            </code>
            <button className="btn btn-primary" onClick={() => handleCopy(createdSecret, 'new_created')}>
              {copiedKeyID === 'new_created' ? 'Copied!' : 'Copy Key'}
            </button>
            <button className="btn btn-secondary" onClick={() => setCreatedSecret(null)}>
              Dismiss
            </button>
          </div>
        </div>
      )}

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
        {loading ? (
          <div style={{ padding: '2rem', textAlign: 'center' }}>Loading API keys from backend...</div>
        ) : keys.length === 0 ? (
          <div style={{ padding: '2rem', textAlign: 'center', color: 'var(--text-muted)' }}>
            No API keys created yet. Generate one above!
          </div>
        ) : (
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
                        onClick={() => handleCopy(k.fullKey, k.id)}
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
        )}
      </div>
    </div>
  )
}

