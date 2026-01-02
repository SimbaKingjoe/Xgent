const API_BASE = '/api/v1'

export interface User {
  id: number
  username: string
  email: string
}

export interface AuthResponse {
  token: string
  expires_at: string
  user: User
}

export interface Workspace {
  id: number
  name: string
  description: string
}

export interface Resource {
  id: number
  workspace_id: number
  name: string
  type: string
  spec: string
}

export interface Task {
  id: number
  title: string
  prompt: string
  status: string
  progress: number
  result?: string
  created_at: string
}

class ApiClient {
  private token: string | null = null

  setToken(token: string) {
    this.token = token
    if (typeof window !== 'undefined') {
      localStorage.setItem('token', token)
    }
  }

  getToken(): string | null {
    if (!this.token && typeof window !== 'undefined') {
      this.token = localStorage.getItem('token')
    }
    return this.token
  }

  clearToken() {
    this.token = null
    if (typeof window !== 'undefined') {
      localStorage.removeItem('token')
    }
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...((options.headers as Record<string, string>) || {}),
    }

    const token = this.getToken()
    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }

    const response = await fetch(`${API_BASE}${endpoint}`, {
      ...options,
      headers,
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Request failed' }))
      throw new Error(error.error || 'Request failed')
    }

    return response.json()
  }

  // Auth
  async register(username: string, email: string, password: string): Promise<AuthResponse> {
    return this.request('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, email, password }),
    })
  }

  async login(username: string, password: string): Promise<AuthResponse> {
    return this.request('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    })
  }

  // Workspaces
  async getWorkspaces(): Promise<Workspace[]> {
    const res = await this.request<{ workspaces: Workspace[] }>('/workspaces')
    return res.workspaces || []
  }

  async createWorkspace(name: string, description: string): Promise<Workspace> {
    return this.request('/workspaces', {
      method: 'POST',
      body: JSON.stringify({ name, description }),
    })
  }

  // Resources
  async getResources(workspaceId?: number): Promise<Resource[]> {
    const query = workspaceId ? `?workspace_id=${workspaceId}` : ''
    const res = await this.request<{ resources: Resource[] }>(`/resources${query}`)
    return res.resources || []
  }

  async createResource(data: { workspace_id: number; name: string; type: string; spec: string }): Promise<Resource> {
    return this.request('/resources', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async updateResource(id: number, data: { name?: string; spec?: string; description?: string }): Promise<Resource> {
    return this.request(`/resources/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async deleteResource(id: number): Promise<void> {
    await this.request(`/resources/${id}`, { method: 'DELETE' })
  }

  // Tasks
  async getTasks(): Promise<Task[]> {
    const res = await this.request<{ tasks: Task[] }>('/tasks')
    return res.tasks || []
  }

  async createTask(data: { title: string; prompt: string; resource_name: string; resource_type: string }): Promise<Task> {
    return this.request('/tasks', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async getTask(id: number): Promise<Task> {
    return this.request(`/tasks/${id}`)
  }
}

export const api = new ApiClient()
