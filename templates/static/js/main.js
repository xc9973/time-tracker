document.addEventListener('DOMContentLoaded', () => {
  const page = document.body.dataset.page
  if (!page) return

  switch (page) {
    case 'sessions':
      initSessionsPage()
      break
  }
})

// Helper function to get API key from page
function getAPIKey() {
  const metaTag = document.querySelector('meta[name="api-key"]')
  return metaTag ? metaTag.getAttribute('content') : ''
}

function initSessionsPage() {
  const startTimeInput = document.getElementById('running-start-time')
  const timerDisplay = document.getElementById('timer-display')

  // Timer logic
  if (startTimeInput && timerDisplay) {
    const startTime = new Date(startTimeInput.value)

    const updateTimer = () => {
      const now = new Date()
      const diff = Math.floor((now - startTime) / 1000)

      if (diff < 0) {
        timerDisplay.textContent = '0:00:00'
        return
      }

      const hours = Math.floor(diff / 3600)
      const minutes = Math.floor((diff % 3600) / 60)
      const seconds = diff % 60

      timerDisplay.textContent = `${hours}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
    }

    updateTimer()
    setInterval(updateTimer, 1000)
  }

  // Session Action Functions
  window.startSession = () => {
    const category = document.getElementById('startCategory').value.trim()
    const task = document.getElementById('startTask').value.trim()
    const note = document.getElementById('startNote').value.trim()

    const baseUrl = window.location.origin;
    fetch(`${baseUrl}/api/v1/sessions/start`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': getAPIKey()
      },
      body: JSON.stringify({ category, task, note }),
      credentials: 'same-origin'
    }).then(response => {
      if (response.ok) {
        window.location.reload()
      } else {
        response.text().then(text => alert('开始计时失败: ' + text))
      }
    }).catch(err => alert('请求错误: ' + err))
  }

  window.stopSession = () => {
    if (!confirm('确定结束当前计时吗？')) return

    const baseUrl = window.location.origin;
    fetch(`${baseUrl}/api/v1/sessions/stop`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': getAPIKey()
      },
      body: JSON.stringify({}),
      credentials: 'same-origin'
    }).then(response => {
      if (response.ok) {
        window.location.reload()
      } else {
        response.text().then(text => alert('结束计时失败: ' + text))
      }
    }).catch(err => alert('请求错误: ' + err))
  }

  window.deleteSession = (id) => {
    if (!confirm('确定删除这条记录吗？此操作无法撤销。')) return

    const baseUrl = window.location.origin;
    fetch(`${baseUrl}/api/v1/sessions/${id}`, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': getAPIKey()
      },
      credentials: 'same-origin'
    }).then(response => {
      if (response.ok) {
        window.location.reload()
      } else {
        response.text().then(text => alert('删除失败: ' + text))
      }
    }).catch(err => alert('请求错误: ' + err))
  }

  // Edit Modal Functions
  window.openEditSession = (btn) => {
    document.getElementById('editId').value = btn.dataset.id
    document.getElementById('editCategory').value = btn.dataset.category
    document.getElementById('editTask').value = btn.dataset.task
    document.getElementById('editNote').value = btn.dataset.note || ''
    document.getElementById('editStart').value = formatForInput(btn.dataset.start)

    // Check if end time exists and format it, otherwise clear input
    const end = btn.dataset.end
    document.getElementById('editEnd').value = end ? formatForInput(end) : ''

    document.getElementById('editModal').style.display = 'flex'
  }

  window.closeEditModal = () => {
    document.getElementById('editModal').style.display = 'none'
  }

  window.saveEditSession = () => {
    const id = document.getElementById('editId').value
    const category = document.getElementById('editCategory').value.trim()
    const task = document.getElementById('editTask').value.trim()
    const note = document.getElementById('editNote').value.trim()
    const startedAt = document.getElementById('editStart').value
    const endedAt = document.getElementById('editEnd').value

    if (!category || !task || !startedAt) {
      alert('请填写必要信息（分类、任务、开始时间）')
      return
    }

    const payload = {
      category,
      task,
      note,
      start_time: toRFC3339(startedAt)
    }

    if (endedAt) {
      payload.end_time = toRFC3339(endedAt)
    }

    const baseUrl = window.location.origin;
    fetch(`${baseUrl}/api/v1/sessions/${id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': getAPIKey()
      },
      body: JSON.stringify(payload),
      credentials: 'same-origin'
    }).then(response => {
      if (response.ok) {
        window.location.reload()
      } else {
        response.text().then(text => alert('保存失败: ' + text))
      }
    }).catch(err => alert('请求错误: ' + err))
  }

  // Attach event listeners to buttons
  const startBtn = document.getElementById('startSessionBtn')
  if (startBtn) {
    startBtn.addEventListener('click', window.startSession)
  }

  const stopBtn = document.getElementById('stopSessionBtn')
  if (stopBtn) {
    stopBtn.addEventListener('click', window.stopSession)
  }

  const cancelEditBtn = document.getElementById('cancelEditBtn')
  if (cancelEditBtn) {
    cancelEditBtn.addEventListener('click', window.closeEditModal)
  }

  const saveEditBtn = document.getElementById('saveEditBtn')
  if (saveEditBtn) {
    saveEditBtn.addEventListener('click', window.saveEditSession)
  }

  // Event delegation for edit and delete buttons
  const tableContainer = document.querySelector('.table-container')
  if (tableContainer) {
    tableContainer.addEventListener('click', (e) => {
      // Handle edit button (and its children)
      const editBtn = e.target.closest('.btn-edit')
      if (editBtn) {
        window.openEditSession(editBtn)
        return
      }

      // Handle delete button (and its children)
      const deleteBtn = e.target.closest('.btn-delete')
      if (deleteBtn) {
        window.deleteSession(deleteBtn.dataset.id)
        return
      }
    })
  }

  // Close modal when clicking outside
  const editModal = document.getElementById('editModal')
  if (editModal) {
    editModal.addEventListener('click', (e) => {
      if (e.target === editModal) window.closeEditModal()
    })
  }
}

// Helper Functions
function formatForInput(isoStr) {
  if (!isoStr) return ''
  const date = new Date(isoStr)
  // Adjust for timezone offset to show correct local time in input
  const offset = date.getTimezoneOffset() * 60000
  return new Date(date - offset).toISOString().slice(0, 16)
}

function toRFC3339(localStr) {
  if (!localStr) return null
  const date = new Date(localStr)
  return date.toISOString()
}
