import React, { useState } from 'react'

export default function PreferencesView() {
  const [recipientID, setRecipientID] = useState('usr_alice')
  const [quietHoursEnabled, setQuietHoursEnabled] = useState(true)
  const [startUTC, setStartUTC] = useState('22:00')
  const [endUTC, setEndUTC] = useState('07:00')

  const [channels, setChannels] = useState({
    webhook: true,
    push: true,
    email: false,
    sms: true,
    in_app: true,
  })

  const toggleChannel = (ch) => {
    setChannels({ ...channels, [ch]: !channels[ch] })
  }

  return (
    <div>
      <div className="section-header">
        <div>
          <h1 className="section-title">User Preference Inspector</h1>
          <p className="section-desc">
            Inspect and override recipient DND quiet hours, timezone configurations, and channel opt-in rules.
          </p>
        </div>
        <button className="btn btn-primary">Save Recipient Preferences</button>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1.5rem' }}>
        <div className="glass-card">
          <h3 style={{ marginBottom: '1rem', fontSize: '1rem' }}>Recipient Configuration</h3>

          <div style={{ marginBottom: '1.25rem' }}>
            <label style={{ display: 'block', fontSize: '0.8rem', color: 'var(--text-muted)', marginBottom: '0.35rem' }}>
              Search Recipient User ID
            </label>
            <input
              className="input code-font"
              value={recipientID}
              onChange={(e) => setRecipientID(e.target.value)}
            />
          </div>

          <div style={{ borderTop: '1px solid var(--border-color)', paddingTop: '1rem', marginTop: '1rem' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
              <div>
                <strong style={{ fontSize: '0.9rem' }}>Do Not Disturb (Quiet Hours)</strong>
                <p style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>
                  Suppress non-critical dispatches during recipient sleeping window
                </p>
              </div>
              <button
                className={`btn ${quietHoursEnabled ? 'btn-primary' : 'btn-secondary'}`}
                style={{ padding: '0.35rem 0.75rem' }}
                onClick={() => setQuietHoursEnabled(!quietHoursEnabled)}
              >
                {quietHoursEnabled ? 'Enabled' : 'Disabled'}
              </button>
            </div>

            {quietHoursEnabled && (
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
                <div>
                  <label style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>Start Window (UTC)</label>
                  <input
                    className="input code-font"
                    type="time"
                    value={startUTC}
                    onChange={(e) => setStartUTC(e.target.value)}
                  />
                </div>
                <div>
                  <label style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>End Window (UTC)</label>
                  <input
                    className="input code-font"
                    type="time"
                    value={endUTC}
                    onChange={(e) => setEndUTC(e.target.value)}
                  />
                </div>
              </div>
            )}
          </div>
        </div>

        <div className="glass-card">
          <h3 style={{ marginBottom: '1rem', fontSize: '1rem' }}>Channel Opt-In Matrix</h3>
          <p style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginBottom: '1.25rem' }}>
            Configure allowed delivery channels for recipient <strong className="code-font">{recipientID}</strong>.
          </p>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
            {Object.keys(channels).map((ch) => (
              <div
                key={ch}
                style={{
                  display: 'flex',
                  justify: 'space-between',
                  alignItems: 'center',
                  padding: '0.75rem 1rem',
                  background: 'rgba(0,0,0,0.4)',
                  borderRadius: 'var(--radius-md)',
                  border: '1px solid var(--border-color)',
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                  <span className="badge badge-purple">{ch.toUpperCase()}</span>
                </div>
                <button
                  className={`btn ${channels[ch] ? 'btn-primary' : 'btn-secondary'}`}
                  style={{ padding: '0.25rem 0.65rem', fontSize: '0.75rem' }}
                  onClick={() => toggleChannel(ch)}
                >
                  {channels[ch] ? 'Opted-In' : 'Opted-Out'}
                </button>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
