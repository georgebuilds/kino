<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useThemeStore } from './stores/theme'
import AppShell from './components/layout/AppShell.vue'
import SetupView from './views/SetupView.vue'
import { GetFileState } from '../wailsjs/go/main/App'

useThemeStore()

const route = useRoute()
const router = useRouter()
const ready = ref(false)

onMounted(async () => {
  try {
    const state = await GetFileState()
    if (!state.isOpen) router.replace('/setup')
  } catch {
    router.replace('/setup')
  }
  ready.value = true
})
</script>

<template>
  <template v-if="ready">
    <SetupView v-if="route.name === 'setup'" />
    <AppShell v-else />
  </template>
</template>
