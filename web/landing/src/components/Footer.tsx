export default function Footer() {
  return (
    <footer className="mt-8 bg-gray-50">
      <div className="container-narrow py-10 text-center text-sm text-gray-700">
        <p>
          If you like Scriberr, consider giving the project a star on
          {' '}<a href="https://github.com/rishikanthc/scriberr" target="_blank" rel="noreferrer" className="text-blue-600 hover:text-blue-700 underline-offset-2 hover:underline">GitHub</a>.
        </p>
        <div className="mt-4 flex justify-center">
          <a href='https://ko-fi.com/H2H41KQZA3' target='_blank' rel="noopener noreferrer">
            <img height='36' style={{border: '0px', height: '36px'}} src='https://storage.ko-fi.com/cdn/kofi6.png?v=6' alt='Buy Me a Coffee at ko-fi.com' />
          </a>
        </div>
      </div>
    </footer>
  );
}
