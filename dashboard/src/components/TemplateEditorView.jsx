import React, { useState } from 'react'

export default function TemplateEditorView() {
  const [templateName, setTemplateName] = useState('comment_coalesce_v1')
  const [subject, setSubject] = useState('New interactions on your post')
  const [body, setBody] = useState(
    '{{ actor_id }} and {{ count | default: 1 }} others liked your post: "{{ object_id }}"'
  )

  const [sampleActor, setSampleActor] = useState('Alice')
  const [sampleCount, setSampleCount] = useState(14)
  const [sampleObject, setSampleObject] = useState('My Summer Trip')

  const renderedPreview = body
    .replace('{{ actor_id }}', sampleActor)
    .replace('{{ count | default: 1 }}', sampleCount)
    .replace('{{ object_id }}', sampleObject)

  return (
    <div>
      <div className="section-header">
        <div>
          <h1 className="section-title">Liquid Notification Template Editor</h1>
          <p className="section-desc">
            Edit dynamic Liquid template strings and test real-time render preview.
          </p>
        </div>
        <button className="btn btn-primary">Save Template</button>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1.5rem' }}>
        <div className="glass-card">
          <h3 style={{ marginBottom: '1rem', fontSize: '1rem' }}>Template Editor</h3>

          <div style={{ marginBottom: '1rem' }}>
            <label style={{ display: 'block', fontSize: '0.8rem', color: 'var(--text-muted)', marginBottom: '0.35rem' }}>
              Template Key Name
            </label>
            <input
              className="input code-font"
              value={templateName}
              onChange={(e) => setTemplateName(e.target.value)}
            />
          </div>

          <div style={{ marginBottom: '1rem' }}>
            <label style={{ display: 'block', fontSize: '0.8rem', color: 'var(--text-muted)', marginBottom: '0.35rem' }}>
              Subject Line
            </label>
            <input
              className="input"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
            />
          </div>

          <div style={{ marginBottom: '1rem' }}>
            <label style={{ display: 'block', fontSize: '0.8rem', color: 'var(--text-muted)', marginBottom: '0.35rem' }}>
              Body Template (Liquid Syntax)
            </label>
            <textarea
              className="input code-font"
              rows={8}
              value={body}
              onChange={(e) => setBody(e.target.value)}
            />
          </div>
        </div>

        <div className="glass-card">
          <h3 style={{ marginBottom: '1rem', fontSize: '1rem' }}>Live Preview & Mock Scope</h3>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '0.5rem', marginBottom: '1.5rem' }}>
            <div>
              <label style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>actor_id</label>
              <input
                className="input code-font"
                value={sampleActor}
                onChange={(e) => setSampleActor(e.target.value)}
              />
            </div>
            <div>
              <label style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>count</label>
              <input
                className="input code-font"
                type="number"
                value={sampleCount}
                onChange={(e) => setSampleCount(Number(e.target.value))}
              />
            </div>
            <div>
              <label style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>object_id</label>
              <input
                className="input code-font"
                value={sampleObject}
                onChange={(e) => setSampleObject(e.target.value)}
              />
            </div>
          </div>

          <div
            style={{
              background: 'rgba(0,0,0,0.6)',
              border: '1px solid var(--border-color)',
              borderRadius: 'var(--radius-md)',
              padding: '1.25rem',
            }}
          >
            <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginBottom: '0.5rem' }}>
              RENDERED OUTPUT PREVIEW
            </div>
            <div style={{ fontWeight: '700', fontSize: '1rem', marginBottom: '0.5rem', color: 'var(--text-primary)' }}>
              {subject}
            </div>
            <div style={{ color: 'var(--text-secondary)', fontSize: '0.9rem', lineHeight: '1.5' }}>
              {renderedPreview}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
