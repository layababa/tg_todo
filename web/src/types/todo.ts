export type TaskStatus = 'pending' | 'completed'

export interface Person {
  id: string
  displayName: string
  avatarUrl?: string
  username?: string
}

export interface Task {
  id: string
  title: string
  description?: string
  status: TaskStatus
  createdAt: string
  createdBy: Person
  assignees: Person[]
  sourceMessageUrl?: string
  permissions: {
    canEdit: boolean
    canComplete: boolean
    canDelete: boolean
  }
}

export interface UpdateTaskPayload {
  title?: string
  status?: TaskStatus
}
