import React, { useEffect, useState } from 'react'

export default function MetricsView() {
  const [metrics, setMetrics] = useState({
    throughput: 1420,
    p99Latency: 14.2,
    activeWS: 842,
    queueDepth: 3,
  })

  useEffect(() => {
    const interval = setInterval(() => {
      setMetrics({
        throughput: Math.floor(1380 + Math.random() * 120),
        p99Latency: +(12.5 + Math.random() * 4).toFixed(1),
        activeWS: 840 + Math.floor(Math.random() * 10),
        queueDepth: Math.floor(Math.random() * 8),
      })
    }, 2000)
    return () => clearInterval(interval)
  }, [])

  return (
    <div>
      <div className="section-header">
        <div>
          <h1 className="section-title">Real-Time Observability</h1>
          <p className="section-desc">
            Live ingestion, fanout latency, active WebSocket connections, and queue depth metrics.
          </p>
        </div>
        <span className="badge badge-emerald">Live Stream Active</span>
      </div>

      <div className="grid-metrics">
        <div className="glass-card">
          <div className="metric-title">Event Ingestion Rate</div>
          <div className="metric-value">{metrics.throughput.toLocaleString()} req/s</div>
          <div className="metric-sub">↑ 4.2% vs last hour</div>
        </div>

        <div className="glass-card">
          <div className="metric-title">p99 Fanout Latency</div>
          <div className="metric-value">{metrics.p99Latency} ms</div>
          <div className="metric-sub" style={{ color: 'var(--accent-emerald)' }}>
            Target: &lt; 50 ms
          </div>
        </div>

        <div className="glass-card">
          <div className="metric-title">Active WebSocket Clients</div>
          <div className="metric-value">{metrics.activeWS}</div>
          <div className="metric-sub">Connected via Redis Pub/Sub</div>
        </div>

        <div className="glass-card">
          <div className="metric-title">NATS Queue Depth</div>
          <div className="metric-value">{metrics.queueDepth}</div>
          <div className="metric-sub">Worker Pool Capacity: 100%</div>
        </div>
      </div>

      <div className="glass-card" style={{ marginTop: '1.5rem' }}>
        <h3 style={{ marginBottom: '1rem', fontSize: '1.1rem' }}>
          Fanout Distribution & Celebrity Cache Hit Ratio
        </h3>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '2rem' }}>
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.85rem', marginBottom: '0.4rem' }}>
              <span>Push Write-Path ZSET Fanout</span>
              <span className="code-font">88.4%</span>
            </div>
            <div style={{ height: '8px', background: 'rgba(255,255,255,0.08)', borderRadius: '4px', overflow: 'hidden' }}>
              <div style={{ width: '88.4%', height: '100%', background: 'var(--primary)' }}></div>
            </div>

            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.85rem', marginTop: '1rem', marginBottom: '0.4rem' }}>
              <span>Pull Celebrity Read-Path Merge</span>
              <span className="code-font">11.6%</span>
            </div>
            <div style={{ height: '8px', background: 'rgba(255,255,255,0.08)', borderRadius: '4px', overflow: 'hidden' }}>
              <div style={{ width: '11.6%', height: '100%', background: 'var(--accent-cyan)' }}></div>
            </div>
          </div>

          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.85rem', marginBottom: '0.4rem' }}>
              <span>Celebrity Post Redis Cache Hit Rate</span>
              <span className="code-font">99.1%</span>
            </div>
            <div style={{ height: '8px', background: 'rgba(255,255,255,0.08)', borderRadius: '4px', overflow: 'hidden' }}>
              <div style={{ width: '99.1%', height: '100%', background: 'var(--accent-emerald)' }}></div>
            </div>

            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.85rem', marginTop: '1rem', marginBottom: '0.4rem' }}>
              <span>DND Quiet Hours Filtered Rate</span>
              <span className="code-font">3.2%</span>
            </div>
            <div style={{ height: '8px', background: 'rgba(255,255,255,0.08)', borderRadius: '4px', overflow: 'hidden' }}>
              <div style={{ width: '3.2%', height: '100%', background: 'var(--accent-purple)' }}></div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
