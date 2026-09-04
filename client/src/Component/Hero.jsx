import React from 'react'

const Hero = () => {
  return (
    <div className="min-h-screen bg-[#f8f9fc] px-5 py-12 sm:px-6 sm:py-16 lg:py-20">
      <div className="mx-auto w-full max-w-2xl text-center">

        <h1 className="text-[32px] font-bold leading-tight text-[#292929] sm:text-2xl lg:text-[48px]">Video Calls, Made Simple</h1>

          <p className="mx-auto mt-4 max-w-xl text-sm leading-6 text-gray-500 sm:text-base sm:leading-7">Connect with anyone, anywhere. NO complicated setup required. Just Press Create and start talking.</p>

          <div className='mx-auto mt-6 flex w-full max-w-sm flex-col gap-3'>

            <button className='h-12 w-full rounded-full bg-[#3f2bd4] text-sm font-semibold text-white transition hover:bg-[#3421b8] active:scale-[0.98]'>
              Create a Call
            </button>
              
            <button className='h-12 w-full rounded-full border-gray-800 border-2 bg-transparent text-sm font-semibold transistion hover:bg-amber-50 '>
              Join a Call
            </button>


          </div>

      </div>

    </div>
  )
}

export default Hero
