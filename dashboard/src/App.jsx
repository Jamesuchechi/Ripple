import React, { useState } from 'react'
import './App.css'
import Sidebar from './components/Sidebar'
import Header from './components/Header'
import AuthModal from './components/AuthModal'
import MetricsView from './components/MetricsView'
import DeliveryLogsView from './components/DeliveryLogsView'
import WorkflowBuilderView from './components/WorkflowBuilderView'
import TemplateEditorView from './components/TemplateEditorView'
import DLQInspectorView from './components/DLQInspectorView'
import PreferencesView from './components/PreferencesView'
import APIKeysView from './components/APIKeysView'
import UsageView from './components/UsageView'

const initialProjects = [
  { id: '00000000-0000-0000-0000-000000000001', name: 'Primary Production App' },
  { id: '11111111-2222-3333-4444-555555555555', name: 'Staging Ecosystem' },
]

function App() {
  const [activeView, setActiveView] = useState('metrics')
  const [projects] = useState(initialProjects)
  const [activeProjectID, setActiveProjectID] = useState(initialProjects[0].id)
  const [user, setUser] = useState({ email: 'admin@ripple.dev', id: 'usr_admin' })
  const [isAuthOpen, setIsAuthOpen] = useState(false)

  const renderActiveView = () => {
    switch (activeView) {
      case 'metrics':
        return <MetricsView />
      case 'logs':
        return <DeliveryLogsView />
      case 'workflows':
        return <WorkflowBuilderView />
      case 'templates':
        return <TemplateEditorView />
      case 'dlq':
        return <DLQInspectorView />
      case 'preferences':
        return <PreferencesView />
      case 'apikeys':
        return <APIKeysView />
      case 'usage':
        return <UsageView />
      default:
        return <MetricsView />
    }
  }

  return (
    <div className="dashboard-container">
      <Sidebar activeView={activeView} setActiveView={setActiveView} />
      <div className="main-content">
        <Header
          projects={projects}
          activeProjectID={activeProjectID}
          setActiveProjectID={setActiveProjectID}
          user={user}
          onOpenAuth={() => setIsAuthOpen(true)}
        />
        <main className="content-area">{renderActiveView()}</main>
      </div>

      <AuthModal
        isOpen={isAuthOpen}
        onClose={() => setIsAuthOpen(false)}
        user={user}
        setUser={setUser}
      />
    </div>
  )
}

export default App
