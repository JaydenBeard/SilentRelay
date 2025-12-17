//
//  Item.swift
//  SilentRelay
//
//  Created by Jayden Beard on 15/12/2025.
//

import Foundation
import SwiftData

@Model
final class Item {
    var timestamp: Date
    
    init(timestamp: Date) {
        self.timestamp = timestamp
    }
}
