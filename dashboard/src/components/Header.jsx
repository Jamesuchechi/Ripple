import React from 'react'

export default function Header({
  projects,
  activeProjectID,
  setActiveProjectID,
  user,
  onOpenAuth,
}) {
  return (
    <header className="header">
      <div className="header-left">
        <select
          className="project-select"
          value={activeProjectID}
          onChange={(e) => setActiveProjectID(e.target.value)}
        >
          {projects.map((p) => (
            <option key={p.id} value={p.id}>
              📁 {p.name} ({p.id.substring(0, 8)}...)
            </option>
          ))}
        </select>
        <span className="badge badge-purple">PRO TIER</span>
      </div>

      <div className="header-right">
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <div className="status-dot"></div>
          <span style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
            Engine Operational
          </span>
        </div>

        {user ? (
          <button className="btn btn-secondary" onClick={onOpenAuth}>
            👤 {user.email}
          </button>
        ) : (
          <button className="btn btn-primary" onClick={onOpenAuth}>
            Sign In
          </button>
        )}
      </div>
    </header>
  )
}
