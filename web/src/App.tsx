import './App.css'
import { createBrowserRouter, RouterProvider } from "react-router-dom"
import Page from './app/page'

const router = createBrowserRouter([
  {
    path: "*",
    element: <Page />,
  }
]);

function App() {
  return (
    <RouterProvider router={router} />
  )
}

export default App
