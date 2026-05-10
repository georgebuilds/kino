import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import { usePreferredColorScheme } from '@vueuse/core'

export type ThemePreference = 'system' | 'light' | 'dark'

const STORAGE_KEY = 'kino-theme'

export const useThemeStore = defineStore('theme', () => {
  const systemScheme = usePreferredColorScheme()
  const preference = ref<ThemePreference>(
    (localStorage.getItem(STORAGE_KEY) as ThemePreference) ?? 'system'
  )

  function resolvedTheme(): 'light' | 'dark' {
    if (preference.value === 'system') {
      return systemScheme.value === 'dark' ? 'dark' : 'light'
    }
    return preference.value
  }

  function applyTheme() {
    document.documentElement.setAttribute('data-theme', resolvedTheme())
  }

  function setPreference(p: ThemePreference) {
    preference.value = p
    localStorage.setItem(STORAGE_KEY, p)
  }

  // Apply on system change or preference change
  watch([preference, systemScheme], applyTheme, { immediate: true })

  return { preference, setPreference, resolvedTheme }
})
