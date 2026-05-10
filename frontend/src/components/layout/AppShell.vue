<template>
  <div class="shell">
    <AppSidebar :collapsed="sidebarCollapsed" @toggle="sidebarCollapsed = !sidebarCollapsed" />
    <main class="shell__main">
      <RouterView v-slot="{ Component }">
        <Transition name="page" mode="out-in">
          <component :is="Component" />
        </Transition>
      </RouterView>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import AppSidebar from './AppSidebar.vue'
import { useAccountsStore } from '../../stores/accounts'
import { useCategoriesStore } from '../../stores/categories'

const sidebarCollapsed = ref(false)

// Eagerly load accounts + categories — used across many views.
const accountsStore   = useAccountsStore()
const categoriesStore = useCategoriesStore()

onMounted(() => {
  accountsStore.fetch()
  categoriesStore.fetch()
})
</script>

<style scoped>
.shell {
  display: flex;
  width: 100%;
  height: 100vh;
  overflow: hidden;
}

.shell__main {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.page-enter-active,
.page-leave-active {
  transition: opacity var(--duration-base) var(--ease-out);
}

.page-enter-from,
.page-leave-to {
  opacity: 0;
}
</style>
