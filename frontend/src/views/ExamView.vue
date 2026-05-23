<template>
  <div>
    <div v-if="loading" class="spinner"></div>
    <div v-else-if="error" class="alert alert-error">{{ error }}</div>

    <div v-else>
      <div class="card">
        <label class="field-label">ชื่อผู้สอบ <span style="color:var(--danger)">*</span></label>
        <input
          v-model="examinee"
          class="field-input"
          placeholder="กรอกชื่อ-นามสกุล"
          :disabled="submitted"
        />
      </div>

      <div style="margin-bottom:16px;">
        <div style="display:flex;justify-content:space-between;font-size:.85rem;color:var(--muted);margin-bottom:4px;">
          <span>ตอบแล้ว {{ answeredCount }} / {{ questions.length }} ข้อ</span>
          <span>{{ Math.round(answeredCount / questions.length * 100) || 0 }}%</span>
        </div>
        <div class="progress-bar">
          <div
            class="progress-fill"
            :style="{ width: (answeredCount / questions.length * 100) + '%' }"
          ></div>
        </div>
      </div>

      <div
        v-for="(q, idx) in questions"
        :key="q.id"
        class="q-card"
        :class="{ answered: answers[q.id] != null }"
      >
        <div class="q-number">ข้อที่ {{ idx + 1 }}</div>
        <div class="q-body">{{ q.body }}</div>
        <div class="choice-list">
          <div
            v-for="c in q.choices"
            :key="c.id"
            class="choice-item"
            :class="{ selected: answers[q.id] === c.id }"
            @click="!submitted && selectAnswer(q.id, c.id)"
          >
            <div class="choice-radio">
              <div v-if="answers[q.id] === c.id" class="choice-radio-dot"></div>
            </div>
            <span>{{ c.body }}</span>
          </div>
        </div>
      </div>

      <div v-if="submitError" class="alert alert-error">{{ submitError }}</div>

      <div style="text-align:center;margin-top:8px;">
        <button
          class="btn btn-primary"
          style="min-width:200px;font-size:1.05rem;"
          :disabled="submitting || answeredCount === 0 || !examinee.trim()"
          @click="submit"
        >
          <span v-if="submitting">กำลังส่ง...</span>
          <span v-else>ส่งข้อสอบ</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { fetchQuestions, submitExam } from '../api.js'

const emit = defineEmits(['submitted'])

const questions   = ref([])
const examinee    = ref('')
const answers     = reactive({}) // { [questionId]: choiceId }
const loading     = ref(true)
const error       = ref('')
const submitting  = ref(false)
const submitError = ref('')
const submitted   = ref(false)

const answeredCount = computed(() =>
  Object.keys(answers).filter(k => answers[k] != null).length
)

onMounted(async () => {
  try {
    questions.value = await fetchQuestions()
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
})

function selectAnswer(qid, cid) {
  answers[qid] = cid
}

async function submit() {
  submitError.value = ''
  if (!examinee.value.trim()) {
    submitError.value = 'กรุณากรอกชื่อผู้สอบ'
    return
  }
  if (answeredCount.value < questions.value.length) {
    const missing = questions.value.length - answeredCount.value
    if (!confirm(`ยังมีข้อที่ยังไม่ได้ตอบอีก ${missing} ข้อ ต้องการส่งข้อสอบเลยหรือไม่?`)) return
  }

  submitting.value = true
  submitted.value  = true
  try {
    const result = await submitExam(examinee.value.trim(), answers)
    emit('submitted', result)
  } catch (e) {
    submitError.value = e.message
    submitted.value   = false
  } finally {
    submitting.value = false
  }
}
</script>
