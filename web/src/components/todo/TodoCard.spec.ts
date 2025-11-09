import { fireEvent, render } from '@testing-library/vue'
import { describe, expect, it, vi } from 'vitest'

import type { Task } from '@/types/todo'
import TodoCard from './TodoCard.vue'

const baseTask: Task = {
  id: '1',
  title: '测试任务',
  status: 'pending',
  createdAt: new Date().toISOString(),
  createdBy: { id: '1', displayName: '创建人' },
  assignees: [],
  permissions: { canEdit: true, canComplete: true, canDelete: true }
}

describe('TodoCard', () => {
  it('emits select when clicked', async () => {
    const { emitted, getByText } = render(TodoCard, {
      props: {
        task: baseTask
      }
    })

    await fireEvent.click(getByText('测试任务'))
    expect(emitted().select?.[0]).toEqual(['1'])
  })

  it('emits toggle when checkbox changes', async () => {
    const onToggle = vi.fn()
    const { getByRole } = render(TodoCard, {
      props: {
        task: baseTask
      },
      attrs: {
        onToggle
      }
    })

    await fireEvent.click(getByRole('checkbox'))
    expect(onToggle).toHaveBeenCalledWith('1', 'completed')
  })
})
