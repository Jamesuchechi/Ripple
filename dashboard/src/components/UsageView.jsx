import React from 'react'

export default function UsageView() {
  return (
    <div>
      <div className="section-header">
        <div>
          <h1 className="section-title">Usage & Billing Control Plane</h1>
          <p className="section-desc">
            Monitor ingested event volumes, channel dispatches, and plan quotas.
          </p>
        </div>
        <span className="badge badge-purple">Pro Plan Active</span>
      </div>

      <div className="grid-metrics" style={{ marginBottom: '1.5rem' }}>
        <div className="glass-card">
          <div className="metric-title">Monthly Ingested Events</div>
          <div className="metric-value">2,450,000</div>
          <div className="metric-sub">Quota Limit: 5,000,000 / mo</div>
        </div>

        <div className="glass-card">
          <div className="metric-title">Total Channel Dispatches</div>
          <div className="metric-value">9,812,400</div>
          <div className="metric-sub">Avg Fanout Multiplier: 4.0x</div>
        </div>

        <div className="glass-card">
          <div className="metric-title">Active WebSocket Peak</div>
          <div className="metric-value">12,450</div>
          <div className="metric-sub">Unlimited Connections</div>
        </div>

        <div className="glass-card">
          <div className="metric-title">Current Billing Period</div>
          <div className="metric-value">24 Days Left</div>
          <div className="metric-sub">Renews Oct 1, 2026</div>
        </div>
      </div>

      <div className="glass-card">
        <h3 style={{ marginBottom: '1rem', fontSize: '1rem' }}>Plan Usage Meter</h3>
        <div style={{ marginBottom: '1rem' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.5rem' }}>
            <span style={{ fontWeight: 600 }}>Monthly Event Quota</span>
            <span className="code-font" style={{ color: 'var(--accent-cyan)' }}>49% Used (2.45M / 5.0M)</span>
          </div>
          <div style={{ height: '12px', background: 'rgba(255,255,255,0.08)', borderRadius: '6px', overflow: 'hidden' }}>
            <div
              style={{
                width: '49%',
                height: '100%',
                background: 'linear-gradient(90deg, var(--primary) 0%, var(--accent-cyan) 100%)',
              }}
            ></div>
          </div>
        </div>

        <div style={{ marginTop: '2rem', borderTop: '1px solid var(--border-color)', paddingTop: '1rem' }}>
          <h4 style={{ fontSize: '0.9rem', marginBottom: '0.75rem' }}>Channel Dispatch Distribution</h4>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))', gap: '1rem' }}>
            <div style={{ background: 'rgba(0,0,0,0.4)', padding: '1rem', borderRadius: '8px', border: '1px solid var(--border-color)' }}>
              <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>Webhooks</div>
              <div style={{ fontSize: '1.25rem', fontWeight: 700, marginTop: '0.2rem' }}>4.2M</div>
            </div>
            <div style={{ background: 'rgba(0,0,0,0.4)', padding: '1rem', borderRadius: '8px', border: '1px solid var(--border-color)' }}>
              <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>Push (FCM/APNs)</div>
              <div style={{ fontSize: '1.25rem', fontWeight: 700, marginTop: '0.2rem' }}>3.8M</div>
            </div>
            <div style={{ background: 'rgba(0,0,0,0.4)', padding: '1rem', borderRadius: '8px', border: '1px solid var(--border-color)' }}>
              <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>Email (SendGrid)</div>
              <div style={{ fontSize: '1.25rem', fontWeight: 700, marginTop: '0.2rem' }}>1.2M</div>
            </div>
            <div style={{ background: 'rgba(0,0,0,0.4)', padding: '1rem', borderRadius: '8px', border: '1px solid var(--border-color)' }}>
              <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>SMS (Twilio)</div>
              <div style={{ fontSize: '1.25rem', fontWeight: 700, marginTop: '0.2rem' }}>612K</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
