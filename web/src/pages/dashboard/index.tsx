import { useEffect, useState } from 'react'
import Layout from '../../components/Layout'
import { api, Workspace, Task } from '../../lib/api'

export default function DashboardPage() {
  const [workspaces, setWorkspaces] = useState<Workspace[]>([])
  const [tasks, setTasks] = useState<Task[]>([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [ws, ts] = await Promise.all([
        api.getWorkspaces(),
        api.getTasks(),
      ])
      setWorkspaces(ws)
      setTasks(ts)
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const createWorkspace = async () => {
    setCreating(true)
    try {
      await api.createWorkspace('Default Workspace', 'My first workspace')
      loadData()
    } catch (err: any) {
      alert(err.message)
    } finally {
      setCreating(false)
    }
  }

  if (loading) {
    return <Layout><div className="text-center py-8">Loading...</div></Layout>
  }

  // Show create workspace prompt if no workspaces exist
  if (workspaces.length === 0) {
    return (
      <Layout>
        <div className="text-center py-16">
          <h1 className="text-2xl font-bold mb-4">Welcome to Xgent! 👋</h1>
          <p className="text-gray-500 mb-6">Create your first workspace to get started</p>
          <button onClick={createWorkspace} disabled={creating} className="btn-primary">
            {creating ? 'Creating...' : '+ Create Workspace'}
          </button>
        </div>
      </Layout>
    )
  }

  return (
    <Layout>
      <h1 className="text-2xl font-bold mb-6">Dashboard</h1>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
        <div className="card">
          <div className="text-3xl font-bold text-indigo-500">{workspaces.length}</div>
          <div className="text-gray-500">Workspaces</div>
        </div>
        <div className="card">
          <div className="text-3xl font-bold text-indigo-500">{tasks.length}</div>
          <div className="text-gray-500">Tasks</div>
        </div>
        <div className="card">
          <div className="text-3xl font-bold text-indigo-500">
            {tasks.filter(t => t.status === 'completed').length}
          </div>
          <div className="text-gray-500">Completed</div>
        </div>
      </div>

      <div className="card">
        <h2 className="text-lg font-semibold mb-4">Recent Tasks</h2>
        {tasks.length === 0 ? (
          <p className="text-gray-500">No tasks yet</p>
        ) : (
          <div className="space-y-3">
            {tasks.slice(0, 5).map((task) => (
              <div key={task.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                <div>
                  <div className="font-medium">{task.title}</div>
                  <div className="text-sm text-gray-500">{task.prompt.slice(0, 50)}...</div>
                </div>
                <span className={`px-2 py-1 rounded text-sm ${
                  task.status === 'completed' ? 'bg-green-100 text-green-700' :
                  task.status === 'running' ? 'bg-blue-100 text-blue-700' :
                  task.status === 'failed' ? 'bg-red-100 text-red-700' :
                  'bg-gray-100 text-gray-700'
                }`}>
                  {task.status}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </Layout>
  )
}
