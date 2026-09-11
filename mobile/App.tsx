import { useEffect } from 'react'
import { StatusBar } from 'expo-status-bar'
import { SafeAreaProvider } from 'react-native-safe-area-context'
import Navigation from './src/navigation'
import { seedBackendUrl } from './src/lib/env'

export default function App() {
  // Mengisi Backend URL dengan nilai awal varian bila masih kosong.
  useEffect(() => {
    seedBackendUrl()
  }, [])

  return (
    <SafeAreaProvider>
      <StatusBar style="dark" />
      <Navigation />
    </SafeAreaProvider>
  )
}
