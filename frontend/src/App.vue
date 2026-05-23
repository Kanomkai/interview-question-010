<template>
  <div>
    <!-- ── Top bar ── -->
    <div class="top-bar">
      <div>
        <div class="logo">📝 ระบบสอบออนไลน์</div>
        <div class="sub">example.com · Interview Question #010</div>
      </div>
    </div>

    <!-- ── Main content ── -->
    <div class="page-wrapper">
      <transition name="fade" mode="out-in">
        <!-- IT 10-1: Exam page -->
        <div v-if="page === 'exam'" key="exam">
          <div style="margin-bottom:20px;">
            <h1 style="font-size:1.5rem;font-weight:700;">แบบทดสอบ</h1>
            <p style="color:var(--muted);font-size:.9rem;margin-top:4px;">
              เลือกคำตอบที่ถูกต้องเพียงข้อเดียวในแต่ละข้อ
            </p>
          </div>
          <ExamView :key="examKey" @submitted="onSubmitted" />
        </div>

        <!-- IT 10-2: Result page -->
        <div v-else key="result">
          <div style="margin-bottom:20px;">
            <h1 style="font-size:1.5rem;font-weight:700;">ผลการสอบ (IT 10-2)</h1>
          </div>
          <ResultView
            :result="lastResult"
            :questions="cachedQuestions"
            @retry="retry"
          />
        </div>
      </transition>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import ExamView   from './views/ExamView.vue'
import ResultView from './views/ResultView.vue'
import { fetchQuestions } from './api.js'

const page           = ref('exam')
const lastResult     = ref(null)
const cachedQuestions = ref([])
const examKey        = ref(0)   // force re-mount on retry

async function onSubmitted(result) {
  // Cache questions so ResultView can show them for review
  try { cachedQuestions.value = await fetchQuestions() } catch { /**/ }
  lastResult.value = result
  page.value = 'result'
}

function retry() {
  examKey.value++
  page.value = 'exam'
}
</script>
