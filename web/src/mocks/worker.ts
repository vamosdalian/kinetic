import { setupWorker } from 'msw/browser'
import { handlers } from './handlers';
const worker = setupWorker()
worker.use(...handlers)

export { worker }