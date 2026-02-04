use std::collections::HashMap;
use std::time::Instant;

// Request represents an incoming HTTP request
struct Request {
    id: i32,
    path: String,
    headers: HashMap<String, String>,
    body: Vec<u8>,
}

// Response represents the processed response
struct Response {
    status_code: i32,
    body: String,
    headers: HashMap<String, String>,
}

// ProcessingContext holds temporary data for request processing
struct ProcessingContext {
    parsed_params: HashMap<String, String>,
    temp_buffers: Vec<Vec<u8>>,
    metadata: Vec<String>,
}

// Rust equivalent - ownership handles memory automatically
fn process_request(req: &Request) -> Response {
    // Stack-allocated, automatically cleaned up when function returns
    let mut ctx = ProcessingContext {
        parsed_params: HashMap::new(),
        temp_buffers: Vec::with_capacity(10),
        metadata: Vec::with_capacity(5),
    };

    // Simulate processing with temporary allocations
    for i in 0..100 {
        ctx.temp_buffers.push(vec![0u8; 1024]);
        ctx.metadata.push(format!("meta-{}", i));
    }

    // ctx is dropped here automatically - all memory freed
    // No GC needed, no manual free needed, guaranteed safe

    // Return response (ownership transferred to caller)
    Response {
        status_code: 200,
        body: format!("Processed request {}", req.id),
        headers: {
            let mut headers = HashMap::new();
            headers.insert("Content-Type".to_string(), "text/plain".to_string());
            headers
        },
    }
    // ctx and all its allocations are automatically freed here
}

// Example with explicit lifetime management
fn process_request_with_borrowed_buffer<'a>(
    req: &Request,
    shared_buffer: &'a mut Vec<u8>,
) -> Response {
    // Can reuse buffer across requests - compiler ensures safety
    shared_buffer.clear();
    shared_buffer.extend_from_slice(b"temporary data");

    // Compiler ensures shared_buffer outlives this function
    Response {
        status_code: 200,
        body: format!("Processed request {} with buffer reuse", req.id),
        headers: HashMap::new(),
    }
}

fn benchmark<F>(name: &str, mut process_fn: F)
where
    F: FnMut(&Request) -> Response,
{
    let start = Instant::now();

    for i in 0..10000 {
        let req = Request {
            id: i,
            path: "/api/users".to_string(),
            headers: {
                let mut h = HashMap::new();
                h.insert("User-Agent".to_string(), "Rust".to_string());
                h
            },
            body: b"request body".to_vec(),
        };

        let resp = process_fn(&req);
        // resp automatically freed here
        drop(resp); // Explicit drop (though automatic anyway)
    }

    let elapsed = start.elapsed();
    println!("{}: {:?}", name, elapsed);
}

fn main() {
    println!("Rust Ownership-Based Memory Management");
    println!("Processing 10,000 requests...\n");

    benchmark("Rust (standard)", process_request);

    // Example with buffer reuse
    let mut shared_buffer = Vec::with_capacity(1024);
    let start = Instant::now();
    for i in 0..10000 {
        let req = Request {
            id: i,
            path: "/api/users".to_string(),
            headers: HashMap::new(),
            body: b"request body".to_vec(),
        };
        let _resp = process_request_with_borrowed_buffer(&req, &mut shared_buffer);
    }
    println!("Rust (buffer reuse): {:?}", start.elapsed());

    println!("\nKey features:");
    println!("- Memory automatically freed when values go out of scope (RAII)");
    println!("- No garbage collector - deterministic performance");
    println!("- Compile-time guarantees prevent use-after-free");
    println!("- Lifetimes ensure borrowed data remains valid");
    println!("- Zero-cost abstractions - no runtime overhead");
}
