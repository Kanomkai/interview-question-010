const BASE = '/api'

export async function fetchQuestions() {
  const res = await fetch(`${BASE}/questions`)
  if (!res.ok) throw new Error('Failed to load questions')
  return res.json()
}

export async function submitExam(examinee, answers) {
  const res = await fetch(`${BASE}/submit`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ examinee, answers }),
  })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'Failed to submit exam')
  }
  return res.json()
}

export async function fetchResults() {
  const res = await fetch(`${BASE}/results`)
  if (!res.ok) throw new Error('Failed to load results')
  return res.json()
}
