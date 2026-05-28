from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Optional
import uuid

app = FastAPI(title="CloudMart Product Service")

class Product(BaseModel):
    id: Optional[str] = None
    name: str
    description: str
    price: float
    stock: int

products = [
    Product(id="1", name="Laptop Pro", description="16GB RAM 512GB SSD", price=999.99, stock=50),
    Product(id="2", name="Wireless Mouse", description="Ergonomic 2.4GHz", price=29.99, stock=200),
    Product(id="3", name="Mechanical Keyboard", description="TKL RGB backlit", price=79.99, stock=150),
]

@app.get("/health")
def health():
    return {"status": "ok", "service": "product-service"}

@app.get("/products", response_model=List[Product])
def list_products():
    return products

@app.get("/products/{product_id}", response_model=Product)
def get_product(product_id: str):
    p = next((p for p in products if p.id == product_id), None)
    if not p:
        raise HTTPException(status_code=404, detail="Product not found")
    return p

@app.post("/products", response_model=Product, status_code=201)
def create_product(product: Product):
    product.id = str(uuid.uuid4())
    products.append(product)
    return product

@app.delete("/products/{product_id}")
def delete_product(product_id: str):
    global products
    before = len(products)
    products = [p for p in products if p.id != product_id]
    if len(products) == before:
        raise HTTPException(status_code=404, detail="Product not found")
    return {"deleted": product_id}
