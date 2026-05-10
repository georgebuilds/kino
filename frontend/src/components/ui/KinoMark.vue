<!--
  Kino — elk head silhouette, left-facing profile.
  Nose points lower-left, antler rack upper-right.
  evenodd fill cuts the eye as negative space so it
  reads against any background colour.
-->
<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 64 64"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    aria-label="Kino"
  >
    <!--
      Head + neck silhouette.
      Path traces clockwise from nose tip.
      Ear is a detour between forehead and antler base.
      Inner eye circle uses evenodd rule for negative space.
    -->
    <path
      fill="currentColor"
      fill-rule="evenodd"
      d="
        M 8 52
        C 6 47, 6 39, 10 32
        C 13 26, 18 20, 24 16
        C 27 13, 30 11, 33 11
        C 32 8, 31 4, 31 4
        C 33 4, 35 8, 34 11
        C 37 10, 40 10, 42 12
        C 46 14, 49 20, 49 28
        C 49 36, 46 44, 42 50
        C 38 56, 30 61, 20 60
        C 12 59, 9 57, 8 52
        Z
        M 21 28
        m -2.5 0
        a 2.5 2.5 0 1 0 5 0
        a 2.5 2.5 0 1 0 -5 0
      "
    />

    <!-- Antler rack — stroked from skull, same colour as fill -->
    <!-- Main beam: sweeps up and back from skull -->
    <path
      stroke="currentColor"
      stroke-width="2.5"
      stroke-linecap="round"
      stroke-linejoin="round"
      fill="none"
      d="M 42 12 C 47 7, 51 4, 54 2"
    />
    <!-- Brow tine: forward-pointing (left, toward nose) -->
    <path
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      fill="none"
      d="M 45 9 L 39 3"
    />
    <!-- Second tine: upward from mid-beam -->
    <path
      stroke="currentColor"
      stroke-width="1.8"
      stroke-linecap="round"
      fill="none"
      d="M 50 6 L 56 7"
    />
    <!-- Crown fork at beam tip -->
    <path
      stroke="currentColor"
      stroke-width="1.6"
      stroke-linecap="round"
      fill="none"
      d="M 54 2 L 51 0 M 54 2 L 58 1"
    />

    <!-- Antler gold accent at base of neck -->
    <circle
      fill="var(--color-accent)"
      cx="21"
      cy="58"
      r="2"
    />
  </svg>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ size?: number }>(), { size: 32 })
</script>

<style scoped>
svg { color: var(--color-primary); display: block; }
</style>
